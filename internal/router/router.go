package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sheranthaperera93/r2-notify-server/internal/handler"
	"github.com/sheranthaperera93/r2-notify-server/internal/middleware"
	configurationService "github.com/sheranthaperera93/r2-notify-server/internal/services/configuration"
	keyService "github.com/sheranthaperera93/r2-notify-server/internal/services/key"
	notificationService "github.com/sheranthaperera93/r2-notify-server/internal/services/notification"
)

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "r2-notify"})
}

func RegisterRoutes(
	r *gin.Engine,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	keyHandler *handler.KeyHandler,
	notificationHandler *handler.NotificationHandler,
	notifSvc notificationService.NotificationService,
	configSvc configurationService.ConfigurationService,
	keySvc *keyService.KeyService,
) {
	// Health check
	r.GET("/health", healthCheck)

	// WebSocket — API key auth happens inside the handler via first-message handshake
	wsHandler := handler.NewWebSocketHandler(notifSvc, configSvc, keySvc)
	r.GET("/ws", gin.WrapF(wsHandler))

	// Notification publish endpoint — validated by X-API-Key header (Unkey)
	r.POST("/notification", notificationHandler.CreateNotification)

	// --- API v1 ---
	v1 := r.Group("/api/v1")

	// Public auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/verify-email", authHandler.VerifyEmail)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
	}

	// Protected routes — require JWT
	protected := v1.Group("")
	protected.Use(middleware.RequireAuth())
	{
		protected.GET("/user/me", userHandler.Me)

		keys := protected.Group("/keys")
		{
			keys.POST("", keyHandler.CreateKey)
			keys.GET("", keyHandler.ListKeys)
			keys.DELETE("/:keyId", keyHandler.RevokeKey)
			keys.GET("/:keyId", keyHandler.GetKeyDetails)
			keys.PATCH("/:keyId", keyHandler.UpdateKey)
		}
	}
}
