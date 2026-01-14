package router

import (
	"r2-notify/controller"

	"github.com/gin-gonic/gin"
)

func RegisterNotificationRoutes(r *gin.Engine, notificationController *controller.NotificationController) {
	notificationRoute := r.Group("/notification")
	notificationRoute.POST("", notificationController.CreateNotification)
}
