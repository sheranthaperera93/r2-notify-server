package controller

import (
	"fmt"
	"net/http"
	"r2-notify-server/config"
	"r2-notify-server/data"
	"r2-notify-server/logger"
	"r2-notify-server/models"
	clientStore "r2-notify-server/services"
	notificationService "r2-notify-server/services/notification"
	"r2-notify-server/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type NotificationController struct {
	notificationService notificationService.NotificationService
}

// NewNotificationController returns a new instance of NotificationController.
// It requires a notificationService to be injected for its dependencies.
func NewNotificationController(service notificationService.NotificationService) *NotificationController {
	return &NotificationController{notificationService: service}
}

// CreateNotification creates a new notification based on the payload in the request body.
// The request must include the X-User-ID and X-App-ID headers.
// The request body must include the groupKey, message, and status.
// The notification will be sent to the user with the given user ID.
// The response will include the newly created notification.
func (controller *NotificationController) CreateNotification(ctx *gin.Context) {

	authorization := ctx.GetHeader("Authorization")
	appId := ctx.GetHeader("X-App-ID")
	correlationId, _ := ctx.Get(data.CORRELATION_ID)

	bearerPrefix := "Bearer "
	if !strings.HasPrefix(authorization, bearerPrefix) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	tokenString := strings.TrimPrefix(authorization, bearerPrefix)
	userId, err := utils.ValidateToken(tokenString, []byte(config.LoadConfig().JwtSecret))
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "NotificationController",
			Operation:     "CreateNotification",
			Message:       "Invalid token",
			AppId:         appId,
			CorrelationId: correlationId.(string),
			Error:         err,
		})
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	logger.Log.Debug(logger.LogPayload{
		Component:     "NotificationController",
		Operation:     "CreateNotification",
		Message:       "CreateNotification called",
		UserId:        userId,
		AppId:         appId,
		CorrelationId: correlationId.(string),
	})

	if userId == "" || appId == "" {
		logger.Log.Error(logger.LogPayload{
			Component:     "NotificationController",
			Operation:     "CreateNotification",
			Message:       "Missing X-User-ID or X-App-ID header",
			UserId:        userId,
			AppId:         appId,
			CorrelationId: correlationId.(string),
		})
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID and X-App-ID headers are required"})
		return
	}

	var payload data.CreateNotificationRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "NotificationController",
			Operation:     "CreateNotification",
			Message:       "Invalid request payload",
			UserId:        userId,
			AppId:         appId,
			CorrelationId: correlationId.(string),
			Error:         err,
		})
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validator.New().Struct(payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	recordId, err := controller.notificationService.Create(m)
	m.Id = recordId

	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component:     "NotificationController",
			Operation:     "CreateNotification",
			Message:       "Failed to create notification",
			UserId:        userId,
			AppId:         appId,
			CorrelationId: correlationId.(string),
			Error:         err,
		})
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Log.Debug(logger.LogPayload{
		Component:     "NotificationController",
		Operation:     "CreateNotification",
		Message:       fmt.Sprintf("Notification created with payload %v", m),
		UserId:        userId,
		AppId:         appId,
		CorrelationId: correlationId.(string),
	})

	logger.Log.Debug(logger.LogPayload{
		Component:     "NotificationController",
		Operation:     "CreateNotification",
		Message:       "Sending notification to user",
		UserId:        userId,
		AppId:         appId,
		CorrelationId: correlationId.(string),
	})

	clientStore.SendNotificationToUser(data.EventNotification{
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
	ctx.JSON(http.StatusCreated, m)
}
