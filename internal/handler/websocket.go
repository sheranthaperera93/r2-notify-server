package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sheranthaperera93/r2-notify-server/internal/config"
	"github.com/sheranthaperera93/r2-notify-server/internal/data"
	"github.com/sheranthaperera93/r2-notify-server/internal/logger"
	"github.com/sheranthaperera93/r2-notify-server/internal/models"
	clientService "github.com/sheranthaperera93/r2-notify-server/internal/services/client"
	configurationService "github.com/sheranthaperera93/r2-notify-server/internal/services/configuration"
	notificationService "github.com/sheranthaperera93/r2-notify-server/internal/services/notification"
	"github.com/sheranthaperera93/r2-notify-server/internal/utils"

	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	notificationService  notificationService.NotificationService
	configurationService configurationService.ConfigurationService
	redisClient          *redis.Client
}

func NewWebSocketHandler(
	notificationService notificationService.NotificationService,
	configurationService configurationService.ConfigurationService,
	redisClient *redis.Client,
) *WebSocketHandler {
	return &WebSocketHandler{
		notificationService:  notificationService,
		configurationService: configurationService,
		redisClient:          redisClient,
	}
}

// HandleConnection upgrades the HTTP connection to a WebSocket connection, validates
// the short-lived WS token, and manages the client lifecycle.
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	origins := config.LoadConfig().AllowedOrigins
	allowedOrigins := utils.ProcessAllowedOrigins(origins)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return slices.Contains(allowedOrigins, origin)
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Message:   "Upgrade error. Allowed origins: " + fmt.Sprint(allowedOrigins) + ". Received Origin: " + c.Request.Header.Get("Origin"),
			Component: "WebSocket",
			Operation: "HandleConnection",
			Error:     err,
		})
		return
	}

	// Validate and consume the short-lived WS token
	wsToken := c.Query("token")
	if wsToken == "" {
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "missing token"))
		conn.Close()
		return
	}

	userId, err := h.redisClient.GetDel(c, "wstoken:"+wsToken).Result()
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Message:   "Failed to validate WS token",
			Component: "WebSocket",
			Operation: "HandleConnection",
			Error:     err,
		})
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid or expired token"))
		conn.Close()
		return
	}

	// Set pong handler to keep connection alive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		logger.Log.Debug(logger.LogPayload{
			Component: "WebSocket Pong Handler",
			Operation: "SetPongHandler",
			Message:   "Pong received from client " + userId,
			UserId:    userId,
		})
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start pinging client every 30 seconds
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			logger.Log.Debug(logger.LogPayload{
				Component: "WebSocket Ping Handler",
				Operation: "PingHandler",
				Message:   "Ping sent to client " + userId,
				UserId:    userId,
			})
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Log.Error(logger.LogPayload{
					Component: "WebSocket Ping Handler",
					Operation: "PingHandler",
					Message:   "Ping failed for client " + userId,
					UserId:    userId,
					Error:     err,
				})
				clientService.RemoveConnection(userId, conn)
				return
			}
		}
	}()

	correlationId := utils.GenerateUUID()

	// Handle notification configuration
	isEnableNotification := true
	logger.Log.Info(logger.LogPayload{
		Component:     "WebSocket Configuration Handler",
		Operation:     "User Configuration Fetch",
		Message:       "Fetching configuration for client " + userId,
		UserId:        userId,
		CorrelationId: correlationId,
	})
	configuration, err := h.configurationService.FindByAppAndUser(userId)
	if err != nil {
		_, err = h.configurationService.Create(models.Configuration{
			UserId:              userId,
			EnableNotifications: isEnableNotification,
		})
		logger.Log.Info(logger.LogPayload{
			Component:     "WebSocket Configuration Handler",
			Operation:     "User Configuration Create",
			Message:       "Creating configuration for client " + userId,
			UserId:        userId,
			CorrelationId: correlationId,
		})
		if err != nil {
			logger.Log.Error(logger.LogPayload{
				Component:     "WebSocket Configuration Handler",
				Operation:     "User Configuration Create",
				Message:       "Failed to create configuration for client " + userId,
				Error:         err,
				UserId:        userId,
				CorrelationId: correlationId,
			})
			conn.Close()
			return
		}
	} else {
		isEnableNotification = configuration.Data.EnableNotification
	}

	info := models.ClientInfo{
		ID:                 userId,
		ConnectedAt:        time.Now(),
		EnableNotification: isEnableNotification,
	}

	if err := clientService.StoreClient(info, conn); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Redis Store",
			Operation:     "Redis Store Client",
			Message:       "Failed to store client in Redis for client " + userId,
			UserId:        userId,
			Error:         err,
			CorrelationId: correlationId,
		})
		conn.Close()
		return
	}

	logger.Log.Info(logger.LogPayload{
		Component:     "WebSocket",
		Operation:     "HandleConnection",
		Message:       fmt.Sprintf("Client %s connected successfully", userId),
		UserId:        userId,
		CorrelationId: correlationId,
	})

	h.sendAllNotificationsToClient(userId, correlationId, false)
	h.sendConfigurationsToClient(userId, correlationId)

	// Read loop — handles incoming events and disconnect
	go func() {
		defer conn.Close()
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				logger.Log.Info(logger.LogPayload{
					Component:     "WebSocket",
					Operation:     "HandleConnection",
					Message:       fmt.Sprintf("Client %s disconnected", userId),
					UserId:        userId,
					CorrelationId: correlationId,
				})
				clientService.RemoveConnection(userId, conn)
				break
			}

			if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				continue
			}

			if len(message) == 0 {
				continue
			}

			var event data.Event
			if err := json.Unmarshal(message, &event); err != nil {
				logger.Log.Error(logger.LogPayload{
					Component:     "WebSocket Event Handler",
					Operation:     "ParseEvent",
					Message:       "Invalid event format",
					Error:         err,
					UserId:        userId,
					CorrelationId: correlationId,
				})
				continue
			}

			logger.Log.Debug(logger.LogPayload{
				Component:     "WebSocket Event Handler",
				Operation:     "HandleEvent",
				Message:       "Processing event: " + event.Event,
				UserId:        userId,
				CorrelationId: correlationId,
			})

			switch event.Event {
			case data.MARK_AS_READ:
				h.markAsReadAction(userId, correlationId)
			case data.MARK_APP_AS_READ:
				h.markAppReadAction(message, userId, correlationId)
			case data.MARK_GROUP_AS_READ:
				h.markGroupAsReadAction(message, userId, correlationId)
			case data.MARK_NOTIFICATION_AS_READ:
				h.markNotificationAsReadAction(message, userId, correlationId)
			case data.DELETE_NOTIFICATIONS:
				h.deleteNotificationsAction(userId, correlationId)
			case data.DELETE_APP_NOTIFICATIONS:
				h.deleteAppNotificationsAction(message, userId, correlationId)
			case data.DELETE_GROUP_NOTIFICATIONS:
				h.deleteGroupNotificationAction(message, userId, correlationId)
			case data.DELETE_NOTIFICATION:
				h.deleteNotificationAction(message, userId, correlationId)
			case data.RELOAD_NOTIFICATIONS:
				h.sendAllNotificationsToClient(userId, correlationId, false)
			case data.SET_NOTIFICATION_STATUS:
				h.setNotificationStatusAction(message, userId, correlationId)
			default:
				logger.Log.Warn(logger.LogPayload{
					Component:     "WebSocket Event Handler",
					Operation:     "HandleEvent",
					Message:       "Unknown event type: " + event.Event,
					UserId:        userId,
					CorrelationId: correlationId,
				})
			}
		}
	}()
}

func (h *WebSocketHandler) sendAllNotificationsToClient(clientId string, correlationId string, bypassStatusCheck bool) {
	notifications, err := h.notificationService.FindAll(clientId)
	payload := data.NotificationList{
		Event: data.Event{Event: data.LIST_NOTIFICATIONS},
		Data:  notifications,
	}
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Notification Handler",
			Operation:     "FetchNotifications",
			Message:       "Failed to fetch notifications for client " + clientId,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Notification Handler",
		Operation:     "SendNotifications",
		Message:       "Sending all notifications to client: " + clientId,
		CorrelationId: correlationId,
	})
	if err := clientService.SendNotificationListToUser(clientId, payload, bypassStatusCheck); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Notification Handler",
			Operation:     "SendNotifications",
			Message:       "Failed to send notifications to client " + clientId,
			Error:         err,
			CorrelationId: correlationId,
		})
	}
}

func (h *WebSocketHandler) sendEmptyNotificationListToClient(clientId string, correlationId string, bypassNotificationStatus bool) {
	payload := data.NotificationList{
		Event: data.Event{Event: data.LIST_NOTIFICATIONS},
		Data:  []data.Notification{},
	}
	if err := clientService.SendNotificationListToUser(clientId, payload, bypassNotificationStatus); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Notification Handler",
			Operation:     "SendNotifications",
			Message:       "Failed to send empty notification list to client " + clientId,
			Error:         err,
			CorrelationId: correlationId,
		})
	}
}

func (h *WebSocketHandler) sendConfigurationsToClient(clientId string, correlationId string) {
	configuration, err := h.configurationService.FindByAppAndUser(clientId)
	payload := data.Configuration{
		Event: data.Event{Event: data.LIST_CONFIGURATIONS},
		Data: data.NotificationConfig{
			UserID:             clientId,
			EnableNotification: configuration.Data.EnableNotification,
			Id:                 configuration.Data.Id,
		},
	}
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Configuration Handler",
			Operation:     "FetchConfigurations",
			Message:       "Failed to fetch configurations for client " + clientId,
			UserId:        clientId,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Configuration Handler",
		Operation:     "SendConfigurations",
		Message:       "Sending configurations to client: " + clientId,
		UserId:        clientId,
		CorrelationId: correlationId,
	})
	if err := clientService.SendConfigurationToUser(payload, true); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Configuration Handler",
			Operation:     "SendConfigurations",
			Message:       "Failed to send configurations to client " + clientId,
			UserId:        clientId,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
}

func (h *WebSocketHandler) markAsReadAction(clientID string, correlationId string) {
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Mark As Read Action",
		Operation:     "MarkAllAsRead",
		Message:       "Marking all notifications as read for client: " + clientID,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	if err := h.notificationService.MarkAsRead(clientID); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Mark As Read Action",
			Operation:     "MarkAllAsRead",
			Message:       "Failed to mark all notifications as read for client " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	h.sendAllNotificationsToClient(clientID, correlationId, false)
}

func (h *WebSocketHandler) markAppReadAction(message []byte, clientID string, correlationId string) {
	var event data.EventNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Mark App As Read Event",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Mark App As Read Event",
		Operation:     "MarkAppAsRead",
		Message:       "Marking all notifications for app as read for client: " + clientID + ", App ID: " + event.Data.AppId,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	if err := h.notificationService.MarkAppAsRead(clientID, event.Data.AppId); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Mark App As Read Event",
			Operation:     "MarkAppAsRead",
			Message:       "Failed to mark app as read for client " + clientID + ", App ID: " + event.Data.AppId,
			UserId:        clientID,
			CorrelationId: correlationId,
			AppId:         event.Data.AppId,
			Error:         err,
		})
	}
	h.sendAllNotificationsToClient(clientID, correlationId, false)
}

func (h *WebSocketHandler) markGroupAsReadAction(message []byte, clientID string, correlationId string) {
	var event data.EventNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Mark Group As Read Event",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			AppId:         event.Data.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Mark Group As Read Event",
		Operation:     "MarkGroupAsRead",
		Message:       "Marking group as read for client: " + clientID + ", App ID: " + event.Data.AppId + ", Group Key: " + event.Data.GroupKey,
		UserId:        clientID,
		AppId:         event.Data.AppId,
		CorrelationId: correlationId,
	})
	if err := h.notificationService.MarkGroupAsRead(clientID, event.Data.AppId, event.Data.GroupKey); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Mark Group As Read Event",
			Operation:     "MarkGroupAsRead",
			Message:       "Failed to mark group as read for client " + clientID + ", App ID: " + event.Data.AppId + ", Group Key: " + event.Data.GroupKey,
			UserId:        clientID,
			AppId:         event.Data.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	h.sendAllNotificationsToClient(clientID, correlationId, false)
}

func (h *WebSocketHandler) markNotificationAsReadAction(message []byte, clientID string, correlationId string) {
	var event data.EventNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Mark Notification As Read Event",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Mark Notification As Read Event",
		Operation:     "MarkNotificationAsRead",
		Message:       "Marking notification as read for client: " + clientID + ", Notification ID: " + event.Data.Id,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	if err := h.notificationService.MarkNotificationAsRead(clientID, event.Data.Id); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Mark Notification As Read Event",
			Operation:     "MarkNotificationAsRead",
			Message:       "Failed to mark notification as read for client " + clientID + ", Notification ID: " + event.Data.Id,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	h.sendAllNotificationsToClient(clientID, correlationId, false)
}

func (h *WebSocketHandler) deleteNotificationsAction(clientID string, correlationId string) {
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Delete Notifications Action",
		Operation:     "DeleteAllNotifications",
		Message:       "Deleting notifications for client: " + clientID,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	if err := h.notificationService.DeleteNotifications(clientID); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Delete Notifications Action",
			Operation:     "DeleteAllNotifications",
			Message:       "Failed to delete all notifications for client " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	h.sendAllNotificationsToClient(clientID, correlationId, false)
}

func (h *WebSocketHandler) deleteAppNotificationsAction(message []byte, clientID string, correlationId string) {
	var event data.EventNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Delete App Notifications Event",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			AppId:         event.Data.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Delete App Notifications Event",
		Operation:     "DeleteAppNotifications",
		Message:       "Deleting all notifications for app for client: " + clientID + ", App ID: " + event.Data.AppId,
		UserId:        clientID,
		AppId:         event.Data.AppId,
		CorrelationId: correlationId,
	})
	if err := h.notificationService.DeleteAppNotifications(clientID, event.Data.AppId); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Delete App Notifications Event",
			Operation:     "DeleteAppNotifications",
			Message:       "Failed to delete app notifications for client " + clientID + ", App ID: " + event.Data.AppId,
			UserId:        clientID,
			AppId:         event.Data.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	h.sendAllNotificationsToClient(clientID, correlationId, false)
}

func (h *WebSocketHandler) deleteGroupNotificationAction(message []byte, clientID string, correlationId string) {
	var event data.EventNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Delete Group Notifications Event",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			AppId:         event.Data.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Delete Group Notifications Event",
		Operation:     "DeleteGroupNotifications",
		Message:       "Deleting group notifications for client: " + clientID + ", App ID: " + event.Data.AppId + ", Group Key: " + event.Data.GroupKey,
		UserId:        clientID,
		AppId:         event.Data.AppId,
		CorrelationId: correlationId,
	})
	if err := h.notificationService.DeleteGroupNotifications(clientID, event.Data.AppId, event.Data.GroupKey); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Delete Group Notifications Event",
			Operation:     "DeleteGroupNotifications",
			Message:       "Failed to delete group notifications for client " + clientID + ", App ID: " + event.Data.AppId + ", Group Key: " + event.Data.GroupKey,
			UserId:        clientID,
			AppId:         event.Data.AppId,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	h.sendAllNotificationsToClient(clientID, correlationId, false)
}

func (h *WebSocketHandler) deleteNotificationAction(message []byte, clientID string, correlationId string) {
	var event data.EventNotification
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Delete Notification Event",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	logger.Log.Debug(logger.LogPayload{
		Component:     "WebSocket Delete Notification Event",
		Operation:     "DeleteNotification",
		Message:       "Deleting notification for client: " + clientID + ", Notification ID: " + event.Data.Id,
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	if err := h.notificationService.DeleteNotification(clientID, event.Data.Id); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Delete Notification Event",
			Operation:     "DeleteNotification",
			Message:       "Failed to delete notification for client " + clientID + ", Notification ID: " + event.Data.Id,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	h.sendAllNotificationsToClient(clientID, correlationId, false)
}

func (h *WebSocketHandler) setNotificationStatusAction(message []byte, clientID string, correlationId string) {
	var event data.Configuration
	if err := json.Unmarshal(message, &event); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Toggle Notification Status Event",
			Operation:     "ParseEvent",
			Message:       "Invalid event format",
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
		return
	}
	if err := h.configurationService.Update(models.Configuration{
		UserId:              clientID,
		EnableNotifications: event.Data.EnableNotification,
	}); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "WebSocket Toggle Notification Status Event",
			Operation:     "UpdateConfiguration",
			Message:       "Failed to update configuration for client " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
			Error:         err,
		})
	}
	logger.Log.Info(logger.LogPayload{
		Component:     "WebSocket Toggle Notification Status Event",
		Operation:     "UpdateConfiguration",
		Message:       "Updated configuration for client: " + clientID + ", EnableNotification: " + fmt.Sprintf("%v", event.Data.EnableNotification),
		UserId:        clientID,
		CorrelationId: correlationId,
	})
	clientService.UpdateClientInfo(models.ClientInfo{
		ID:                 clientID,
		EnableNotification: event.Data.EnableNotification,
	})
	if event.Data.EnableNotification {
		logger.Log.Debug(logger.LogPayload{
			Component:     "WebSocket Toggle Notification Status Event",
			Operation:     "SendNotifications",
			Message:       "Sending all notifications to client: " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
		})
		h.sendAllNotificationsToClient(clientID, correlationId, false)
	} else {
		logger.Log.Debug(logger.LogPayload{
			Component:     "WebSocket Toggle Notification Status Event",
			Operation:     "SendNotifications",
			Message:       "Sending empty list to client: " + clientID,
			UserId:        clientID,
			CorrelationId: correlationId,
		})
		h.sendEmptyNotificationListToClient(clientID, correlationId, true)
	}
	h.sendConfigurationsToClient(clientID, correlationId)
}
