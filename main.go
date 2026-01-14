package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"r2-notify/config"
	"r2-notify/controller"
	"r2-notify/data"
	"r2-notify/event-hub/consumer"
	"r2-notify/handlers"
	configurationRepository "r2-notify/repository/configuration"
	notificationRepository "r2-notify/repository/notification"
	"r2-notify/router"
	configurationService "r2-notify/services/configuration"
	notificationService "r2-notify/services/notification"
	"r2-notify/utils"
	"syscall"

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
	// Create Gin router
	r := gin.Default()

	notificationRepository := notificationRepository.NewNotificationRepositoryImpl(mongoDb)
	notificationService, err := notificationService.NewNotificationServiceImpl(notificationRepository, validate)
	if err != nil {
		log.Fatalf("Error initializing notification service: %v", err)
	}
	configurationRepository := configurationRepository.NewConfigurationRepositoryImpl(mongoDb)
	configurationService, err := configurationService.NewConfigurationServiceImpl(configurationRepository, validate)
	if err != nil {
		log.Fatalf("Error initializing configuration service: %v", err)
	}

	// Start Event Hub consumer in a goroutuine to avoid blocking
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := consumer.StartEventHubConsumer(ctx, notificationService); err != nil {
			log.Fatalf("Failed to start consumer: %v", err)
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

	// Run signal listener in own goroutine
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Start HTTP server (blocking)
	port := config.LoadConfig().Port
	log.Println("Server started on port: " + port)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}
