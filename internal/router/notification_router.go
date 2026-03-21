package router

import (
	"github.com/sheranthaperera93/r2-notify-server/internal/controller"

	"github.com/gin-gonic/gin"
)

func RegisterNotificationRoutes(r *gin.Engine, notificationController *controller.NotificationController) {
	notificationRoute := r.Group("/notification")
	notificationRoute.POST("", notificationController.CreateNotification)
}
