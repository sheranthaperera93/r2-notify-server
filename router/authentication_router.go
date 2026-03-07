package router

import (
	"r2-notify-server/controller"

	"github.com/gin-gonic/gin"
)

func RegisterAuthenticationRoutes(r *gin.Engine, authController *controller.AuthenticationController) {
	notificationRoute := r.Group("/auth")
	notificationRoute.POST("google", authController.GoogleAuthHandler)
}
