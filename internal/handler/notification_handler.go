package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sheranthaperera93/r2-notify-server/internal/config"
	"github.com/sheranthaperera93/r2-notify-server/internal/data"
	"github.com/sheranthaperera93/r2-notify-server/internal/logger"
	"github.com/sheranthaperera93/r2-notify-server/internal/models"
	clientService "github.com/sheranthaperera93/r2-notify-server/internal/services/client"
	keyService "github.com/sheranthaperera93/r2-notify-server/internal/services/key"
	notificationService "github.com/sheranthaperera93/r2-notify-server/internal/services/notification"
	"github.com/sheranthaperera93/r2-notify-server/internal/utils"
)

type NotificationHandler struct {
	notifySvc notificationService.NotificationService
	keySvc    *keyService.KeyService
}

func NewNotificationHandler(notifySvc notificationService.NotificationService, keySvc *keyService.KeyService) *NotificationHandler {
	return &NotificationHandler{notifySvc: notifySvc, keySvc: keySvc}
}

func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")
	appId := c.GetHeader("X-App-ID")
	correlationId, _ := c.Get(data.CORRELATION_ID)

	userId, err := h.keySvc.ValidateAPIKey(apiKey)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "NotificationHandler",
			Operation:     "CreateNotification",
			Message:       "Invalid API key",
			AppId:         appId,
			CorrelationId: correlationId.(string),
			Error:         err,
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	if appId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-App-ID header is required"})
		return
	}

	var payload data.CreateNotificationRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validator.New().Struct(payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m := models.Notification{
		UserId:     userId,
		AppId:      appId,
		GroupKey:   payload.GroupKey,
		Message:    payload.Message,
		Status:     payload.Status,
		ReadStatus: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	recordId, err := h.notifySvc.Create(m)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "NotificationHandler",
			Operation:     "CreateNotification",
			Message:       "Failed to create notification",
			UserId:        userId,
			AppId:         appId,
			CorrelationId: fmt.Sprintf("%v", correlationId),
			Error:         err,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	m.Id = recordId

	clientService.SendNotificationToUser(data.EventNotification{
		Event: data.Event{Event: "newNotification"},
		Data: data.Notification{
			Id:        recordId.Hex(),
			UserID:    m.UserId,
			AppId:     m.AppId,
			GroupKey:  m.GroupKey,
			Message:   m.Message,
			Status:    m.Status,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}, false)

	c.JSON(http.StatusCreated, m)
}

func (h *NotificationHandler) IssueWebSocketToken(c *gin.Context) {
	correlationId, _ := c.Get(data.CORRELATION_ID)
	apiKey := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if apiKey == "" {
		c.JSON(401, gin.H{"error": "missing api key"})
		return
	}

	// Validate via Unkey (your existing ValidateAPIKey)
	userId, err := h.keySvc.ValidateAPIKey(apiKey)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "NotificationHandler",
			Operation:     "CreateNotification",
			Message:       "Invalid API key",
			CorrelationId: correlationId.(string),
			Error:         err,
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	token, err := utils.GenerateSecureToken()
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "NotificationHandler",
			Operation:     "Issue WebSocket Token",
			Message:       "Failed to generate secure token",
			CorrelationId: correlationId.(string),
			Error:         err,
		})
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}
	key := "wstoken:" + token

	err = config.RDB.Set(c, key, userId, 30*time.Second).Err()
	if err != nil {
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

	c.JSON(200, gin.H{"token": token})
}
