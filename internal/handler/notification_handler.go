package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sheranthaperera93/r2-notify-server/internal/data"
	"github.com/sheranthaperera93/r2-notify-server/internal/logger"
	"github.com/sheranthaperera93/r2-notify-server/internal/models"
	clientService "github.com/sheranthaperera93/r2-notify-server/internal/services/client"
	keyService "github.com/sheranthaperera93/r2-notify-server/internal/services/key"
	notificationService "github.com/sheranthaperera93/r2-notify-server/internal/services/notification"
)

type NotificationHandler struct {
	notifSvc notificationService.NotificationService
	keySvc   *keyService.KeyService
}

func NewNotificationHandler(notifSvc notificationService.NotificationService, keySvc *keyService.KeyService) *NotificationHandler {
	return &NotificationHandler{notifSvc: notifSvc, keySvc: keySvc}
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

	if userId == "" || appId == "" {
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

	recordId, err := h.notifSvc.Create(m)
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
