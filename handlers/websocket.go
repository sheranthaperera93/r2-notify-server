package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	"r2-notify/config"
	"r2-notify/data"
	"r2-notify/logger"
	"r2-notify/models"
	clientStore "r2-notify/services"
	configurationService "r2-notify/services/configuration"
	notificationService "r2-notify/services/notification"
	"r2-notify/utils"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var allowedOrigins []string

// NewWebSocketHandler creates a new HTTP handler function for handling WebSocket connections.
// It upgrades HTTP connections to WebSocket connections, validates request origins, and manages
// client connections by storing them in the client store. The handler retrieves or creates
// notification configurations for clients, sends notifications and configurations to clients,
// and listens for incoming WebSocket messages to handle various client actions. If a connection
// error occurs or the client disconnects, the connection is closed and removed from the client store.
func NewWebSocketHandler(notificationService notificationService.NotificationService, configurationService configurationService.ConfigurationService) http.HandlerFunc {

	origins := config.LoadConfig().AllowedOrigins
	allowedOrigins = utils.ProcessAllowedOrigins(origins)

	return func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return slices.Contains(allowedOrigins, origin)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Log.Error(logger.LogPayload{
				Message:   "Upgrade error",
				Service:   "WebSocket",
				Component: "NewWebSocketHandler",
				Operation: "NewWebSocketHandler",
				Error:     err,
			})
			return
		}

		clientID := r.URL.Query().Get("userId")
		if clientID == "" {
			logger.Log.Error(logger.LogPayload{
				Message:   "Missing user ID",
				Service:   "WebSocket",
				Component: "NewWebSocketHandler",
				Operation: "NewWebSocketHandler",
				Error:     err,
			})
			conn.Close()
			return
		}

		// Set pong handler to keep connection alive
		conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // initial deadline
		conn.SetPongHandler(func(string) error {
			logger.Log.Info(logger.LogPayload{
				Service:   "WebSocket",
				Component: "PongHandler",
				Operation: "SetPongHandler",
				Message:   "Pong received from client " + clientID,
				UserId:    clientID,
			})
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
					logger.Log.Error(logger.LogPayload{
						Service:   "WebSocket",
						Component: "PongHandler",
						Operation: "PingHandler",
						Message:   "Ping failed for client " + clientID,
						UserId:    clientID,
						Error:     err,
					})
					clientStore.RemoveConnection(clientID, conn)
					return
				}
				logger.Log.Info(logger.LogPayload{
					Service:   "WebSocket",
					Component: "PingHandler",
					Operation: "SetPongHandler",
					Message:   "Ping sent to client " + clientID,
					UserId:    clientID,
				})
				<-ticker.C
			}
		}()

		// Generate correlation ID
		correlationId := utils.GenerateUUID()

		// Handle Enable Notification Configuration
		isEnableNotification := true
		logger.Log.Info(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Configuration Handler",
			Operation:     "User Configuration Fetch",
			Message:       "Fetching configuration for client " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
		})
		configuration, err := configurationService.FindByAppAndUser(clientID)
		if err != nil {
			_, err = configurationService.Create(models.Configuration{
				UserId:              clientID,
				EnableNotifications: isEnableNotification,
			})
			logger.Log.Info(logger.LogPayload{
				Service:       "WebSocket",
				Component:     "Configuration Handler",
				Operation:     "User Configuration Create",
				Message:       "Creating configuration for client " + clientID,
				UserId:        clientID,
				CorrelationId: correlationId,
			})
			if err != nil {
				logger.Log.Error(logger.LogPayload{
					Service:       "WebSocket",
					Component:     "Configuration Handler",
					Operation:     "User Configuration Create",
					Message:       "Failed to create configuration for client " + clientID,
					Error:         err,
					UserId:        clientID,
					CorrelationId: correlationId,
				})
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
			logger.Log.Error(logger.LogPayload{
				Service:       "WebSocket",
				Component:     "Redis Store",
				Operation:     "Redis Store Client",
				Message:       "Failed to store client in Redis for client " + clientID,
				UserId:        clientID,
				Error:         err,
				CorrelationId: correlationId,
			})
			conn.Close()
			return
		}

		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Websocket Store",
			Operation:     "Websocket Store Client",
			Message:       fmt.Sprintf("Client %s connected", clientID),
			UserId:        clientID,
			CorrelationId: correlationId,
		})

		// Fetch and send all notifications for the client
		sendAllNotificationsToClient(notificationService, clientID, correlationId)

		// Send Client Configurations
		sendConfigurationsToClient(configurationService, clientID, correlationId)

		// Connection close if client disconnect or error occurs
		go func() {
			defer conn.Close()
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					logger.Log.Info(logger.LogPayload{
						Service:       "WebSocket",
						Component:     "Websocket Store",
						Operation:     "Websocket Store Client",
						Message:       fmt.Sprintf("Client %s disconnected", clientID),
						UserId:        clientID,
						CorrelationId: correlationId,
					})
					clientStore.RemoveConnection(clientID, conn)
					break
				}

				// Parse events
				var action data.Action
				if err := json.Unmarshal(message, &action); err != nil {
					logger.Log.Error(logger.LogPayload{
						Service:       "WebSocket",
						Component:     "Websocket Event Handler",
						Operation:     "ParseAction",
						Message:       "Invalid event format",
						Error:         err,
						UserId:        clientID,
						CorrelationId: correlationId,
					})
					continue
				}

				logger.Log.Debug(logger.LogPayload{
					Service:       "WebSocket",
					Component:     "Websocket Event Handler",
					Operation:     "HandleAction",
					Message:       "Processing action: " + action.Action,
					UserId:        clientID,
					CorrelationId: correlationId,
				})

				// Handle events
				switch action.Action {
				// Mark as Read Actions
				case data.MARK_AS_READ:
					markAsReadAction(notificationService, clientID, correlationId)
				case data.MARK_APP_AS_READ:
					markAppReadAction(message, notificationService, clientID, correlationId)
				case data.MARK_GROUP_AS_READ:
					markGroupAsReadAction(message, notificationService, clientID, correlationId)
				case data.MARK_NOTIFICATION_AS_READ:
					markNotificationAsReadAction(message, notificationService, clientID, correlationId)

				// Delete Actions
				case data.DELETE_NOTIFICATIONS:
					deleteNotificationsAction(notificationService, clientID, correlationId)
				case data.DELETE_APP_NOTIFICATIONS:
					deleteAppNotificationsAction(message, notificationService, clientID, correlationId)
				case data.DELETE_GROUP_NOTIFICATIONS:
					deleteGroupNotificationAction(message, notificationService, clientID, correlationId)
				case data.DELETE_NOTIFICATION:
					deleteNotificationAction(message, notificationService, clientID, correlationId)

				// Other Actions
				case data.RELOAD_NOTIFICATIONS:
					sendAllNotificationsToClient(notificationService, clientID, correlationId)
				case data.TOGGLE_NOTIFICATION_STATUS:
					toggleNotificationStatusAction(message, configurationService, notificationService, clientID, correlationId)
				default:
					logger.Log.Warn(logger.LogPayload{
						Service:       "WebSocket",
						Component:     "Websocket Event Handler",
						Operation:     "HandleAction",
						Message:       "Unknown event type: " + action.Action,
						UserId:        clientID,
						CorrelationId: correlationId,
					})
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
func sendAllNotificationsToClient(notificationService notificationService.NotificationService, clientId string, correlationId string) {
	notifications, err := notificationService.FindAll(clientId)
	payload := data.NotificationList{
		Action: data.Action{Action: data.LIST_NOTIFICATIONS},
		Data:   notifications,
	}
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "WebSocket",
			Component: "Notification Handler",
			Operation: "FetchNotifications",
			Message:   "Failed to fetch notifications for client " + clientId,
			Error:     err,
		})
	} else {
		logger.Log.Debug(logger.LogPayload{
			Service:   "WebSocket",
			Component: "Notification Handler",
			Operation: "SendNotifications",
			Message:   "Sending all notifications to client: " + clientId,
		})
		if err := clientStore.SendNotificationListToUser(clientId, payload); err != nil {
			logger.Log.Error(logger.LogPayload{
				Service:   "WebSocket",
				Component: "Notification Handler",
				Operation: "SendNotifications",
				Message:   "Failed to send notifications to client " + clientId,
				Error:     err,
			})
		}
	}
}

// sendConfigurationsToClient sends the current configuration of a user to the corresponding client
// identified by the given clientId. If the user is not connected or if the configuration fetch fails,
// the function logs an error and does not attempt to send the configuration. If the configuration is
// successfully sent, it will bypass the notification status check.
func sendConfigurationsToClient(configurationService configurationService.ConfigurationService, clientId string, correlationId string) {
	configuration, err := configurationService.FindByAppAndUser(clientId)
	payload := data.Configuration{
		Action:             data.Action{Action: data.LIST_CONFIGURATIONS},
		UserID:             clientId,
		EnableNotification: configuration.EnableNotification,
		Id:                 configuration.Id,
	}
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Configuration Handler",
			Operation:     "FetchConfigurations",
			Message:       "Failed to fetch configurations for client " + clientId,
			UserId:        clientId,
			CorrelationId: correlationId,
			Error:         err,
		})
	} else {
		logger.Log.Debug(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Configuration Handler",
			Operation:     "SendConfigurations",
			Message:       "Sending configurations to client: " + clientId,
			UserId:        clientId,
			CorrelationId: correlationId,
		})
		if err := clientStore.SendConfigurationToUser(payload, true); err != nil {
			logger.Log.Error(logger.LogPayload{
				Service:       "WebSocket",
				Component:     "Configuration Handler",
				Operation:     "SendConfigurations",
				Message:       "Failed to send configurations to client " + clientId,
				UserId:        clientId,
				CorrelationId: correlationId,
				Error:         err,
			})
		}
	}
}

// markAsReadAction handles the event to mark all notifications as read for a given client.
// It marks all notifications as read and then sends the updated list of notifications back to the client.
// Logs errors if the update operation fails.
func markAsReadAction(notificationService notificationService.NotificationService, clientID string, correlationId string) {
	logger.Log.Debug(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Mark As Read Action",
		Operation:     "MarkAllAsRead",
		Message:       "Marking all notifications as read for client: " + clientID,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	err := notificationService.MarkAsRead(clientID)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Mark As Read Action",
			Operation:     "MarkAllAsRead",
			Message:       "Failed to mark all notifications as read for client " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	sendAllNotificationsToClient(notificationService, clientID, correlationId)
}

// markAppReadAction handles the event to mark all notifications for a specific app as read for a given client.
// It unmarshals the incoming message to extract the appId, then uses the notificationService to update the read status
// of the notifications in the database. If successful, it sends the updated list of notifications back to the client.
// Logs errors if the message format is invalid or if the update operation fails.
func markAppReadAction(message []byte, notificationService notificationService.NotificationService, clientID string, correlationId string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Mark App As Read Action",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Mark App As Read Action",
		Operation:     "MarkAppAsRead",
		Message:       "Marking all notifications for app as read for client: " + clientID + ", App ID: " + event.AppId,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	err := notificationService.MarkAppAsRead(clientID, event.AppId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Mark App As Read Action",
			Operation:     "MarkAppAsRead",
			Message:       "Failed to mark app as read for client " + clientID + ", App ID: " + event.AppId,
			UserId:        clientID,
			CorrelationId: correlationId,
			AppId:         event.AppId,
			Error:         err,
		})
	}
	sendAllNotificationsToClient(notificationService, clientID, correlationId)
}

// markGroupAsReadAction handles the event to mark all notifications with a given appId and groupKey as read for a given client.
// It unmarshals the incoming message to extract the appId and groupKey, then uses the notificationService to
// update the read status of the notifications in the database. If successful, it sends the updated list of
// notifications back to the client. Logs errors if the message format is invalid or if the update operation fails.
func markGroupAsReadAction(message []byte, notificationService notificationService.NotificationService, clientID string, correlationId string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Mark Group As Read Action",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			AppId:         event.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Mark Group As Read Action",
		Operation:     "MarkGroupAsRead",
		Message:       "Marking group as read for client: " + clientID + ", App ID: " + event.AppId + ", Group Key: " + event.GroupKey,
		UserId:        clientID,
		AppId:         event.AppId,
		CorrelationId: correlationId,
	})
	err := notificationService.MarkGroupAsRead(clientID, event.AppId, event.GroupKey)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Mark Group As Read Action",
			Operation:     "MarkGroupAsRead",
			Message:       "Failed to mark group as read for client " + clientID + ", App ID: " + event.AppId + ", Group Key: " + event.GroupKey,
			UserId:        clientID,
			AppId:         event.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	sendAllNotificationsToClient(notificationService, clientID, correlationId)
}

// markNotificationAsReadAction handles the event to mark a specific notification as read for a given client.
// It unmarshals the incoming message to extract the notification ID, then uses the notificationService to
// update the read status of the notification in the database. If successful, it sends the updated list of
// notifications back to the client. Logs errors if the message format is invalid or if the update operation fails.
func markNotificationAsReadAction(message []byte, notificationService notificationService.NotificationService, clientID string, correlationId string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Mark Notification As Read Action",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Mark Notification As Read Action",
		Operation:     "MarkNotificationAsRead",
		Message:       "Marking notification as read for client: " + clientID + ", Notification ID: " + event.Id,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	err := notificationService.MarkNotificationAsRead(clientID, event.Id)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Mark Notification As Read Action",
			Operation:     "MarkNotificationAsRead",
			Message:       "Failed to mark notification as read for client " + clientID + ", Notification ID: " + event.Id,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	sendAllNotificationsToClient(notificationService, clientID, correlationId)
}

// deleteNotificationsAction handles the event to delete all notifications for a given client.
// It uses the notificationService to delete the notifications
// in the database. If successful, it sends the updated list of notifications back to the client.
// Logs errors if the message format is invalid or if the update operation fails.
func deleteNotificationsAction(notificationService notificationService.NotificationService, clientID string, correlationId string) {
	logger.Log.Debug(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Delete Notifications Action",
		Operation:     "DeleteAllNotifications",
		Message:       "Deleting notifications for client: " + clientID,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	err := notificationService.DeleteNotifications(clientID)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Delete Notifications Action",
			Operation:     "DeleteAllNotifications",
			Message:       "Failed to delete all notifications for client " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	sendAllNotificationsToClient(notificationService, clientID, correlationId)
}

// deleteAppNotificationsAction handles the event to delete all notifications for a specific app for a given client.
// It unmarshals the incoming message to extract the appId, then uses the notificationService to delete the notifications
// in the database. If successful, it sends the updated list of notifications back to the client.
// Logs errors if the message format is invalid or if the update operation fails.
func deleteAppNotificationsAction(message []byte, notificationService notificationService.NotificationService, clientID string, correlationId string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Delete App Notifications Action",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			AppId:         event.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Delete App Notifications Action",
		Operation:     "DeleteAppNotifications",
		Message:       "Deleting all notifications for app for client: " + clientID + ", App ID: " + event.AppId,
		UserId:        clientID,
		AppId:         event.AppId,
		CorrelationId: correlationId,
	})
	err := notificationService.DeleteAppNotifications(clientID, event.AppId)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Delete App Notifications Action",
			Operation:     "DeleteAppNotifications",
			Message:       "Failed to delete app notifications for client " + clientID + ", App ID: " + event.AppId,
			UserId:        clientID,
			AppId:         event.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	sendAllNotificationsToClient(notificationService, clientID, correlationId)
}

// deleteGroupNotificationAction handles the event to delete all notifications with a given appId and groupKey for a given client.
// It unmarshals the incoming message to extract the appId and groupKey, then uses the notificationService to
// delete the notifications in the database. If successful, it sends the updated list of
// notifications back to the client. Logs errors if the message format is invalid or if the deletion operation fails.
func deleteGroupNotificationAction(message []byte, notificationService notificationService.NotificationService, clientID string, correlationId string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Delete Group Notifications Action",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			AppId:         event.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Delete Group Notifications Action",
		Operation:     "DeleteGroupNotifications",
		Message:       "Deleting group notifications for client: " + clientID + ", App ID: " + event.AppId + ", Group Key: " + event.GroupKey,
		UserId:        clientID,
		AppId:         event.AppId,
		CorrelationId: correlationId,
	})
	err := notificationService.DeleteGroupNotifications(clientID, event.AppId, event.GroupKey)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Delete Group Notifications Action",
			Operation:     "DeleteGroupNotifications",
			Message:       "Failed to delete group notifications for client " + clientID + ", App ID: " + event.AppId + ", Group Key: " + event.GroupKey,
			UserId:        clientID,
			AppId:         event.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	sendAllNotificationsToClient(notificationService, clientID, correlationId)
}

// deleteNotificationAction handles the event to delete a specific notification for a given client.
// It unmarshals the incoming message to extract the notification ID, then uses the notificationService to
// delete the notification from the database. If successful, it sends the updated list of
// notifications back to the client. Logs errors if the message format is invalid or if the deletion operation fails.
func deleteNotificationAction(message []byte, notificationService notificationService.NotificationService, clientID string, correlationId string) {
	var event data.ActionNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Delete Notification Action",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Delete Notification Action",
		Operation:     "DeleteNotification",
		Message:       "Deleting notification for client: " + clientID + ", Notification ID: " + event.Id,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	err := notificationService.DeleteNotification(clientID, event.Id)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Delete Notification Action",
			Operation:     "DeleteNotification",
			Message:       "Failed to delete notification for client " + clientID + ", Notification ID: " + event.Id,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	sendAllNotificationsToClient(notificationService, clientID, correlationId)
}

// toggleNotificationStatusAction handles the toggle notification status event.
// It unmarshals the incoming message to extract the configuration data, updates the user's
// notification settings in the configuration service, and updates the client information in
// the client store. If notifications are enabled, it sends all notifications to the client.
// Finally, it sends the updated configuration back to the client.
func toggleNotificationStatusAction(message []byte, configurationService configurationService.ConfigurationService, notificationService notificationService.NotificationService, clientID string, correlationId string) {
	var event data.Configuration
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Toggle Notification Status Action",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	err := configurationService.Update(models.Configuration{
		UserId:              clientID,
		EnableNotifications: event.EnableNotification,
	})
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Toggle Notification Status Action",
			Operation:     "UpdateConfiguration",
			Message:       "Failed to update configuration for client " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	logger.Log.Info(logger.LogPayload{
		Service:       "WebSocket",
		Component:     "Toggle Notification Status Action",
		Operation:     "UpdateConfiguration",
		Message:       "Updated configuration for client: " + clientID + ", EnableNotification: " + fmt.Sprintf("%v", event.EnableNotification),
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	clientStore.UpdateClientInfo(models.ClientInfo{
		ID:                 clientID,
		EnableNotification: event.EnableNotification,
	})
	if event.EnableNotification {
		logger.Log.Debug(logger.LogPayload{
			Service:       "WebSocket",
			Component:     "Toggle Notification Status Action",
			Operation:     "SendNotifications",
			Message:       "Sending all notifications to client: " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
		})
		sendAllNotificationsToClient(notificationService, clientID, correlationId)
	}
	// Send updated configuration to client
	sendConfigurationsToClient(configurationService, clientID, correlationId)
}
