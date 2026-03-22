package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sheranthaperera93/r2-notify-server/internal/config"
	"github.com/sheranthaperera93/r2-notify-server/internal/controller"
	"github.com/sheranthaperera93/r2-notify-server/internal/data"
	"github.com/sheranthaperera93/r2-notify-server/internal/event-hub/consumer"
	"github.com/sheranthaperera93/r2-notify-server/internal/handlers"
	"github.com/sheranthaperera93/r2-notify-server/internal/logger"
	"github.com/sheranthaperera93/r2-notify-server/internal/middleware"
	configurationRepository "github.com/sheranthaperera93/r2-notify-server/internal/repository/configuration"
	notificationRepository "github.com/sheranthaperera93/r2-notify-server/internal/repository/notification"
	"github.com/sheranthaperera93/r2-notify-server/internal/router"
	authenticationService "github.com/sheranthaperera93/r2-notify-server/internal/services/authentication"
	configurationService "github.com/sheranthaperera93/r2-notify-server/internal/services/configuration"
	notificationService "github.com/sheranthaperera93/r2-notify-server/internal/services/notification"
	"github.com/sheranthaperera93/r2-notify-server/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/cors"

	"github.com/joho/godotenv"
)

func main() {
	// Only load .env file in local development
	if os.Getenv("ENV") != data.PRODUCTION_ENV {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	// Initiate MongoDB
	mongoDb := config.MongoConnection()
	// Init Redis
	config.InitRedis()
	// Initiate Service
	validate := validator.New()
	// Set gin mode
	if os.Getenv("ENV") == data.PRODUCTION_ENV {
		gin.SetMode(gin.ReleaseMode)
	}
	// Create Gin router
	r := gin.Default()
	r.Use(middleware.CorrelationIDMiddleware())

	logger.Init()
	defer logger.Log.Flush()

	notificationRepository := notificationRepository.NewNotificationRepositoryImpl(mongoDb)
	notificationService, err := notificationService.NewNotificationServiceImpl(notificationRepository, validate)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Main",
			Operation: "NotificationService",
			Message:   "Failed to initialize notification service",
			Error:     err,
		})
		os.Exit(1)
	}
	configurationRepository := configurationRepository.NewConfigurationRepositoryImpl(mongoDb)
	configurationService, err := configurationService.NewConfigurationServiceImpl(configurationRepository, validate)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Main",
			Operation: "ConfigurationService",
			Message:   "Failed to initialize configuration service",
			Error:     err,
		})
		os.Exit(1)
	}

	authenticationService, err := authenticationService.NewAuthenticationServiceImpl()

	// Start Event Hub consumer in a goroutuine to avoid blocking
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := consumer.StartEventHubConsumer(ctx, notificationService); err != nil {
			logger.Log.Error(logger.LogPayload{
				Component: "Main",
				Operation: "EventHubConsumer",
				Message:   "Failed to start Event Hub consumer",
				Error:     err,
			})
			os.Exit(1)
		}
	}()

	// Create Notification Controller
	notificationController := controller.NewNotificationController(notificationService)
	authenticationController := controller.NewAuthController(authenticationService)

	// Register routes
	router.RegisterNotificationRoutes(r, notificationController)
	router.RegisterAuthenticationRoutes(r, authenticationController)

	// Health check route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "github.com/sheranthaperera93/r2-notify-server",
		})
	})

	// Register WebSocket route
	r.GET("/ws", func(c *gin.Context) {
		handlers.NewWebSocketHandler(notificationService, configurationService)(c.Writer, c.Request)
	})

	// Enable CORS for all origins and methods needed for REST/WS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   utils.ProcessAllowedOrigins(config.LoadConfig().AllowedOrigins),
		AllowedMethods:   []string{"POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-App-ID", "X-Correlation-ID", "X-API-Key"},
		AllowCredentials: true,
		Debug:            true,
	}).Handler(r)

	srv := &http.Server{
		Addr:    ":" + config.LoadConfig().Port,
		Handler: corsHandler,
	}

	// Running server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
			logger.Log.Error(logger.LogPayload{
				Component: "Main",
				Operation: "ListenAndServe",
				Message:   "Failed to start server",
				Error:     err,
			})
			os.Exit(1)
		}
	}()

	logger.Log.Info(logger.LogPayload{
		Component: "Main",
		Operation: "Startup",
		Message:   fmt.Sprintf("Server started on port %s", config.LoadConfig().Port),
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Log.Info(logger.LogPayload{
		Component: "Main",
		Operation: "Startup",
		Message:   "Received shutdown signal",
	})
	cancel()

	// Gracefully shutdown HTTP server
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Main",
			Operation: "Startup",
			Message:   "Received shutdown signal",
			Error:     err,
		})
		os.Exit(1)
	}
	logger.Log.Info(logger.LogPayload{
		Component: "Main",
		Operation: "Exit",
		Message:   "Server exited properly",
	})

}
