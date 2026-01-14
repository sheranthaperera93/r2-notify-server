package controller

import (
	"address-book-notification-service/data"
	"address-book-notification-service/models"
	clientStore "address-book-notification-service/services"
	notificationService "address-book-notification-service/services/notification"
	"net/http"
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

	userId := ctx.GetHeader("X-User-ID")
	appId := ctx.GetHeader("X-App-ID")

	if userId == "" || appId == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "X-User-ID and X-App-ID headers are required"})
		return
	}

	var payload data.CreateNotificationRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	clientStore.SendNotificationToUser(data.ActionNotification{
		Action: data.Action{Action: "newNotification"},
		Notification: data.Notification{
			Id:        recordId.Hex(),
			UserID:    m.UserId,
			AppId:     m.AppId,
			GroupKey:  m.GroupKey,
			Message:   m.Message,
			Status:    m.Status,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	})
	ctx.JSON(http.StatusCreated, m)
}
