package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"r2-notify/config"
	"r2-notify/data"
	"r2-notify/models"
	clientStore "r2-notify/services"
	configurationService "r2-notify/services/configuration"
	notificationService "r2-notify/services/notification"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var (
	allowedOriginsMap map[string]struct{}
	allowAllOrigins   bool
)

// init initializes the allowedOriginsMap and the allowAllOrigins flag based on the
// configuration's AllowedOrigins setting. If AllowedOrigins is empty, a default set
// of origins is used. The function processes each origin, trimming whitespace, and
// populates the allowedOriginsMap unless a wildcard "*" is found, in which case
// allowAllOrigins is set to true, allowing all origins.
func init() {
	origins := config.LoadConfig().AllowedOrigins
	if origins == "" {
		origins = "http://127.0.0.1:4200,http://localhost:4200"
	}
	allowedOriginsMap = make(map[string]struct{})
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o == "*" {
			allowAllOrigins = true
			break
		}
		if o != "" {
			allowedOriginsMap[o] = struct{}{}
		}
	}
}

// isOriginAllowed returns true if the given origin is allowed to connect to the WebSocket endpoint.
// The configuration option AllowedOrigins is checked, and if it contains "*" then all origins are
// allowed. Otherwise, the origin is checked against the map of allowed origins constructed from the
// configuration option.
func isOriginAllowed(origin string) bool {
	if allowAllOrigins {
		return true
	}
	_, ok := allowedOriginsMap[origin]
	return ok
}

// NewWebSocketHandler creates a new HTTP handler function for handling WebSocket connections.
// It upgrades HTTP connections to WebSocket connections, validates request origins, and manages
// client connections by storing them in the client store. The handler retrieves or creates
// notification configurations for clients, sends notifications and configurations to clients,
// and listens for incoming WebSocket messages to handle various client actions. If a connection
// error occurs or the client disconnects, the connection is closed and removed from the client store.
func NewWebSocketHandler(notificationService notificationService.NotificationService, configurationService configurationService.ConfigurationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return isOriginAllowed(origin)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Upgrade error: %v", err.Error())
			return
		}

		clientID := r.URL.Query().Get("userId")
		if clientID == "" {
			log.Printf("Missing user ID")
			conn.Close()
			return
		}

		// Set pong handler to keep connection alive
		conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // initial deadline
		conn.SetPongHandler(func(string) error {
			log.Printf("Received pong from client %s", clientID)
			conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // reset on pong
			return nil
		})

		// Start pinging client every 30 seconds
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer func() {
				ticker.Stop()
				conn.Close()
			}()
			for {
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Ping failed for client %s: %v\n", clientID, err.Error())
					clientStore.RemoveConnection(clientID, conn)
					return
				}
				log.Printf("Sent ping to client %s", clientID)
				<-ticker.C
			}
		}()

		// Handle Enable Notification Configuration
		isEnableNotification := true
		log.Printf("Fetching configuration for client: %s", clientID)
		configuration, err := configurationService.FindByAppAndUser(clientID)
		if err != nil {
			_, err = configurationService.Create(models.Configuration{
				UserId:              clientID,
				EnableNotifications: isEnableNotification,
			})
			log.Printf("Created configuration for client: %s", clientID)
			if err != nil {
				log.Printf("Create configuration error: %v", err.Error())
				conn.Close()
				return
			}
		} else {
			isEnableNotification = configuration.EnableNotification
		}

		info := models.ClientInfo{
			ID:                 clientID,
			ConnectedAt:        time.Now(),
			EnableNotification: isEnableNotification,
		}

		if err := clientStore.StoreClient(info, conn); err != nil {
			log.Printf("Redis store error: %v", err.Error())
			conn.Close()
			return
		}

		log.Printf("Client connected: %s\n", clientID)

		// Fetch and send all notifications for the client
		sendAllNotificationsToClient(notificationService, clientID)

		// Send Client Configurations
		sendConfigurationsToClient(configurationService, clientID)

		// Connection close if client disconnect or error occurs
		go func() {
			defer conn.Close()
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("Client disconnected: %s\n", clientID)
					clientStore.RemoveConnection(clientID, conn)
					break
				}

				// Parse events
				var action data.Action
				if err := json.Unmarshal(message, &action); err != nil {
					log.Printf("Invalid event format: %v", err.Error())
					continue
				}

				// Handle events
				switch action.Action {
				// Mark as Read Actions
				case data.MARK_AS_READ:
					markAsReadAction(notificationService, clientID)
				case data.MARK_APP_AS_READ:
					markAppReadAction(message, notificationService, clientID)
				case data.MARK_GROUP_AS_READ:
					markGroupAsReadAction(message, notificationService, clientID)
				case data.MARK_NOTIFICATION_AS_READ:
					markNotificationAsReadAction(message, notificationService, clientID)

				// Delete Actions
				case data.DELETE_NOTIFICATIONS:
					deleteNotificationsAction(notificationService, clientID)
				case data.DELETE_APP_NOTIFICATIONS:
					deleteAppNotificationsAction(message, notificationService, clientID)
				case data.DELETE_GROUP_NOTIFICATIONS:
					deleteGroupNotificationAction(message, notificationService, clientID)
				case data.DELETE_NOTIFICATION:
					deleteNotificationAction(message, notificationService, clientID)

				// Other Actions
				case data.RELOAD_NOTIFICATIONS:
					sendAllNotificationsToClient(notificationService, clientID)
				case data.TOGGLE_NOTIFICATION_STATUS:
					toggleNotificationStatusAction(message, configurationService, notificationService, clientID)
				default:
					log.Printf("Unknown event type: %s", action.Action)
				}
			}
		}()
	}
}

// sendAllNotificationsToClient sends all the notifications of a user to the corresponding client identified by the given clientId.
// It first fetches all the notifications of the user using the notificationService, then constructs a payload of type NotificationList
// encapsulating the notifications. If the fetch operation fails, it logs an error and does not send the notifications. If the fetch
// operation is successful, it sends the constructed payload to the client using the clientStore. If the send operation fails, it logs
// an error.
func sendAllNotificationsToClient(notificationService notificationService.NotificationService, clientId string) {
	notifications, err := notificationService.FindAll(clientId)
	payload := data.NotificationList{
		Action: data.Action{Action: data.LIST_NOTIFICATIONS},
		Data:   notifications,
	}
	if err != nil {
		log.Printf("Failed to fetch notifications: %v", err.Error())
	} else {
		log.Printf("Sending all notifications to client: %s", clientId)
		if err := clientStore.SendNotificationListToUser(clientId, payload); err != nil {
			log.Printf("Failed to send notifications: %v", err.Error())
		}
	}
}

// sendConfigurationsToClient sends the current configuration of a user to the corresponding client
// identified by the given clientId. If the user is not connected or if the configuration fetch fails,
// the function logs an error and does not attempt to send the configuration. If the configuration is
// successfully sent, it will bypass the notification status check.
func sendConfigurationsToClient(configurationService configurationService.ConfigurationService, clientId string) {
	configuration, err := configurationService.FindByAppAndUser(clientId)
	payload := data.Configuration{
		Action:             data.Action{Action: data.LIST_CONFIGURATIONS},
		UserID:             clientId,
		EnableNotification: configuration.EnableNotification,
		Id:                 configuration.Id,
	}
	if err != nil {
		log.Printf("Failed to fetch configurations: %v", err.Error())
	} else {
		log.Printf("Sending configurations to client: %s", clientId)
		if err := clientStore.SendConfigurationToUser(payload, true); err != nil {
			log.Printf("Failed to send configurations: %v", err.Error())
		}
	}
}

// markAsReadAction handles the event to mark all notifications as read for a given client.
// It marks all notifications as read and then sends the updated list of notifications back to the client.
// Logs errors if the update operation fails.
func markAsReadAction(notificationService notificationService.NotificationService, clientID string) {
	log.Printf("Marking all notifications as read for client: %s", clientID)
	err := notificationService.MarkAsRead(clientID)
	if err != nil {
		log.Printf("Failed to mark all as read: %v", err.Error())
	}
	sendAllNotificationsToClient(notificationService, clientID)
}

// markAppReadAction handles the event to mark all notifications for a specific app as read for a given client.
// It unmarshals the incoming message to extract the appId, then uses the notificationService to update the read status
// of the notifications in the database. If successful, it sends the updated list of notifications back to the client.
// Logs errors if the message format is invalid or if the update operation fails.
func markAppReadAction(message []byte, notificationService notificationService.NotificationService, clientID string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid event format: %v", err.Error())
		return
	}
	log.Printf("Marking all notifications for app as read for client: %s, App ID: %s", clientID, event.AppId)
	err := notificationService.MarkAppAsRead(clientID, event.AppId)
	if err != nil {
		log.Printf("Failed to mark as read: %v", err.Error())
	}
	sendAllNotificationsToClient(notificationService, clientID)
}

// markGroupAsReadAction handles the event to mark all notifications with a given appId and groupKey as read for a given client.
// It unmarshals the incoming message to extract the appId and groupKey, then uses the notificationService to
// update the read status of the notifications in the database. If successful, it sends the updated list of
// notifications back to the client. Logs errors if the message format is invalid or if the update operation fails.
func markGroupAsReadAction(message []byte, notificationService notificationService.NotificationService, clientID string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid event format: %v", err.Error())
		return
	}
	log.Printf("Marking group as read for client: %s, App ID: %s, Group Key: %s", clientID, event.AppId, event.GroupKey)
	err := notificationService.MarkGroupAsRead(clientID, event.AppId, event.GroupKey)
	if err != nil {
		log.Printf("Failed to mark as read: %v", err.Error())
	}
	sendAllNotificationsToClient(notificationService, clientID)
}

// markNotificationAsReadAction handles the event to mark a specific notification as read for a given client.
// It unmarshals the incoming message to extract the notification ID, then uses the notificationService to
// update the read status of the notification in the database. If successful, it sends the updated list of
// notifications back to the client. Logs errors if the message format is invalid or if the update operation fails.
func markNotificationAsReadAction(message []byte, notificationService notificationService.NotificationService, clientID string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid event format: %v", err.Error())
		return
	}
	log.Printf("Marking notification as read for client: %s, Notification ID: %s", clientID, event.Id)
	err := notificationService.MarkNotificationAsRead(clientID, event.Id)
	if err != nil {
		log.Printf("Failed to mark as read: %v", err.Error())
	}
	sendAllNotificationsToClient(notificationService, clientID)
}

// deleteNotificationsAction handles the event to delete all notifications for a given client.
// It uses the notificationService to delete the notifications
// in the database. If successful, it sends the updated list of notifications back to the client.
// Logs errors if the message format is invalid or if the update operation fails.
func deleteNotificationsAction(notificationService notificationService.NotificationService, clientID string) {
	log.Printf("Deleting notifications for client: %s", clientID)
	err := notificationService.DeleteNotifications(clientID)
	if err != nil {
		log.Printf("Failed to delete all notifications: %v", err.Error())
	}
	sendAllNotificationsToClient(notificationService, clientID)
}

// deleteAppNotificationsAction handles the event to delete all notifications for a specific app for a given client.
// It unmarshals the incoming message to extract the appId, then uses the notificationService to delete the notifications
// in the database. If successful, it sends the updated list of notifications back to the client.
// Logs errors if the message format is invalid or if the update operation fails.
func deleteAppNotificationsAction(message []byte, notificationService notificationService.NotificationService, clientID string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid event format: %v", err.Error())
		return
	}
	log.Printf("Deleting all notifications for app as read for client: %s, App ID: %s", clientID, event.AppId)
	err := notificationService.DeleteAppNotifications(clientID, event.AppId)
	if err != nil {
		log.Printf("Failed to delete app notifications: %v", err.Error())
	}
	sendAllNotificationsToClient(notificationService, clientID)
}

// deleteGroupNotificationAction handles the event to delete all notifications with a given appId and groupKey for a given client.
// It unmarshals the incoming message to extract the appId and groupKey, then uses the notificationService to
// delete the notifications in the database. If successful, it sends the updated list of
// notifications back to the client. Logs errors if the message format is invalid or if the deletion operation fails.
func deleteGroupNotificationAction(message []byte, notificationService notificationService.NotificationService, clientID string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid event format: %v", err.Error())
		return
	}
	log.Printf("Deleting group notifications for client: %s, App ID: %s, Group Key: %s", clientID, event.AppId, event.GroupKey)
	err := notificationService.DeleteGroupNotifications(clientID, event.AppId, event.GroupKey)
	if err != nil {
		log.Printf("Failed to delete group notifications: %v", err.Error())
	}
	sendAllNotificationsToClient(notificationService, clientID)
}

// deleteNotificationAction handles the event to delete a specific notification for a given client.
// It unmarshals the incoming message to extract the notification ID, then uses the notificationService to
// delete the notification from the database. If successful, it sends the updated list of
// notifications back to the client. Logs errors if the message format is invalid or if the deletion operation fails.
func deleteNotificationAction(message []byte, notificationService notificationService.NotificationService, clientID string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid event format: %v", err.Error())
		return
	}
	log.Printf("Deleting notification for client: %s, Notification ID: %s", clientID, event.Id)
	err := notificationService.DeleteNotification(clientID, event.Id)
	if err != nil {
		log.Printf("Failed to delete notification: %v", err.Error())
	}
	sendAllNotificationsToClient(notificationService, clientID)
}

// toggleNotificationStatusAction handles the toggle notification status event.
// It unmarshals the incoming message to extract the configuration data, updates the user's
// notification settings in the configuration service, and updates the client information in
// the client store. If notifications are enabled, it sends all notifications to the client.
// Finally, it sends the updated configuration back to the client.
func toggleNotificationStatusAction(message []byte, configurationService configurationService.ConfigurationService, notificationService notificationService.NotificationService, clientID string) {
	var event data.Configuration
	if err := json.Unmarshal(message, &event); err != nil {
		log.Printf("Invalid event format: %v", err.Error())
		return
	}
	err := configurationService.Update(models.Configuration{
		UserId:              clientID,
		EnableNotifications: event.EnableNotification,
	})
	if err != nil {
		log.Printf("Failed to update configuration: %v", err.Error())
	}
	log.Printf("Updated configuration for client: %s", clientID)
	clientStore.UpdateClientInfo(models.ClientInfo{
		ID:                 clientID,
		EnableNotification: event.EnableNotification,
	})
	if event.EnableNotification {
		log.Printf("Sending all notifications to client: %s", clientID)
		sendAllNotificationsToClient(notificationService, clientID)
	}
	// Send updated configuration to client
	sendConfigurationsToClient(configurationService, clientID)
}
