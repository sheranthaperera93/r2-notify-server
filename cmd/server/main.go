package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/cors"
	"github.com/sheranthaperera93/r2-notify-server/internal/config"
	"github.com/sheranthaperera93/r2-notify-server/internal/data"
	"github.com/sheranthaperera93/r2-notify-server/internal/handler"
	"github.com/sheranthaperera93/r2-notify-server/internal/logger"
	"github.com/sheranthaperera93/r2-notify-server/internal/middleware"
	configurationRepository "github.com/sheranthaperera93/r2-notify-server/internal/repository/configuration"
	notificationRepository "github.com/sheranthaperera93/r2-notify-server/internal/repository/notification"
	tokenRepository "github.com/sheranthaperera93/r2-notify-server/internal/repository/token"
	userRepository "github.com/sheranthaperera93/r2-notify-server/internal/repository/user"
	"github.com/sheranthaperera93/r2-notify-server/internal/router"
	authService "github.com/sheranthaperera93/r2-notify-server/internal/services/auth"
	configurationService "github.com/sheranthaperera93/r2-notify-server/internal/services/configuration"
	emailService "github.com/sheranthaperera93/r2-notify-server/internal/services/email"
	keyService "github.com/sheranthaperera93/r2-notify-server/internal/services/key"
	notificationService "github.com/sheranthaperera93/r2-notify-server/internal/services/notification"
	"github.com/sheranthaperera93/r2-notify-server/internal/utils"
)

func main() {
	cfg := config.LoadConfig()

	logger.Init()
	defer logger.Log.Flush()

	// --- Infrastructure ---
	mongoDB := config.MongoConnection()
	config.InitRedis()

	// --- Repositories ---
	userRepo := userRepository.NewUserRepository(mongoDB)
	tokenRepo := tokenRepository.NewTokenRepository(mongoDB)
	notifRepo := notificationRepository.NewNotificationRepositoryImpl(mongoDB)
	configRepo := configurationRepository.NewConfigurationRepositoryImpl(mongoDB)

	// --- Services ---
	validate := validator.New()

	emailSvc := emailService.NewEmailService(cfg.ResendAPIKey, cfg.ResendFrom)
	authSvc := authService.NewAuthService(userRepo, tokenRepo, emailSvc)
	keySvc := keyService.NewKeyService(userRepo)

	notifySvc, err := notificationService.NewNotificationServiceImpl(notifRepo, validate)
	if err != nil {
		logger.Log.Error(logger.LogPayload{Component: "Main", Operation: "Startup", Message: "Failed to init notification service", Error: err})
		os.Exit(1)
	}

	configSvc, err := configurationService.NewConfigurationServiceImpl(configRepo, validate)
	if err != nil {
		logger.Log.Error(logger.LogPayload{Component: "Main", Operation: "Startup", Message: "Failed to init configuration service", Error: err})
		os.Exit(1)
	}

	// --- Handlers ---
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userRepo)
	keyHandler := handler.NewKeyHandler(keySvc)
	notifyHandler := handler.NewNotificationHandler(notifySvc, keySvc)

	// --- Gin ---
	if cfg.Env == data.PRODUCTION_ENV {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationIDMiddleware())

	router.RegisterRoutes(r, authHandler, userHandler, keyHandler, notifyHandler, notifySvc, configSvc, keySvc, config.RDB)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   utils.ProcessAllowedOrigins(cfg.AllowedOrigins),
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Correlation-ID", "X-App-ID", "X-API-Key"},
		AllowCredentials: true,
	}).Handler(r)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: corsHandler,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %s", err)
		}
	}()

	logger.Log.Info(logger.LogPayload{
		Component: "Main",
		Operation: "Startup",
		Message:   "r2-notify started on port " + cfg.Port,
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Log.Info(logger.LogPayload{Component: "Main", Operation: "Shutdown", Message: "Shutting down gracefully..."})

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error(logger.LogPayload{Component: "Main", Operation: "Shutdown", Message: "Forced shutdown", Error: err})
		os.Exit(1)
	}

	logger.Log.Info(logger.LogPayload{Component: "Main", Operation: "Shutdown", Message: "Server exited cleanly"})
}
