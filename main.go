package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"r2-notify/config"
	"r2-notify/controller"
	"r2-notify/data"
	"r2-notify/event-hub/consumer"
	"r2-notify/handlers"
	"r2-notify/logger"
	"r2-notify/middleware"
	configurationRepository "r2-notify/repository/configuration"
	notificationRepository "r2-notify/repository/notification"
	"r2-notify/router"
	configurationService "r2-notify/services/configuration"
	notificationService "r2-notify/services/notification"
	"r2-notify/utils"
	"syscall"
	"time"

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
			Service:   "r2-notify",
			Component: "main",
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
			Service:   "r2-notify",
			Component: "main",
			Operation: "ConfigurationService",
			Message:   "Failed to initialize configuration service",
			Error:     err,
		})
		os.Exit(1)
	}

	// Start Event Hub consumer in a goroutuine to avoid blocking
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := consumer.StartEventHubConsumer(ctx, notificationService); err != nil {
			logger.Log.Error(logger.LogPayload{
				Service:   "r2-notify",
				Component: "main",
				Operation: "EventHubConsumer",
				Message:   "Failed to start Event Hub consumer",
				Error:     err,
			})
			os.Exit(1)
		}
	}()

	// Create Notification Controller
	notificationController := controller.NewNotificationController(notificationService)

	// Register routes
	router.RegisterNotificationRoutes(r, notificationController)

	// Register WebSocket route
	r.GET("/ws", func(c *gin.Context) {
		handlers.NewWebSocketHandler(notificationService, configurationService)(c.Writer, c.Request)
	})

	// Enable CORS for all origins
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   utils.ProcessAllowedOrigins(config.LoadConfig().AllowedOrigins),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-User-ID"},
		AllowCredentials: true,
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
				Service:   "r2-notify",
				Component: "main",
				Operation: "ListenAndServe",
				Message:   "Failed to start server",
				Error:     err,
			})
			os.Exit(1)
		}
	}()

	logger.Log.Info(logger.LogPayload{
		Service:   "r2-notify",
		Component: "main",
		Operation: "Startup",
		Message:   fmt.Sprintf("Server started on port %s", config.LoadConfig().Port),
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Log.Info(logger.LogPayload{
		Service:   "r2-notify",
		Component: "main",
		Operation: "Startup",
		Message:   "Received shutdown signal",
	})
	cancel()

	// Gracefully shutdown HTTP server
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Log.Error(logger.LogPayload{
			Service:   "r2-notify",
			Component: "main",
			Operation: "Startup",
			Message:   "Received shutdown signal",
			Error:     err,
		})
		os.Exit(1)
	}
	logger.Log.Info(logger.LogPayload{
		Service:   "r2-notify",
		Component: "main",
		Operation: "Exit",
		Message:   "Server exited properly",
	})

}
