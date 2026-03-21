package router

import (
	"github.com/sheranthaperera93/r2-notify-server/internal/controller"

	"github.com/gin-gonic/gin"
)

func RegisterAuthenticationRoutes(r *gin.Engine, authController *controller.AuthenticationController) {
	notificationRoute := r.Group("/auth")
	notificationRoute.POST("api-key", authController.ApiKeyAuthHandler)
}
