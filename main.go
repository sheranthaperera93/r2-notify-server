package main

import (
	"address-book-notification-service/config"
	"address-book-notification-service/controller"
	"address-book-notification-service/event-hub/consumer"
	"address-book-notification-service/handlers"
	configurationRepository "address-book-notification-service/repository/configuration"
	notificationRepository "address-book-notification-service/repository/notification"
	"address-book-notification-service/router"
	configurationService "address-book-notification-service/services/configuration"
	notificationService "address-book-notification-service/services/notification"
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/cors"

	"github.com/joho/godotenv"
)

var allowedOrigins []string

func main() {

	// Load origins
	processAllowedOrigins()

	// Only load .env file in local development
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	// Initiate MongoDB
	mongoDb := config.MongoConnection()

	// Initiate Service
	validate := validator.New()
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

	// Init Redis
	config.InitRedis()

	// Start Event Hub consumer in a goroutuine to avoid blocking
	ctx := context.Background()
	go func() {
		if err := consumer.StartEventHubConsumer(ctx, notificationService); err != nil {
			log.Fatalf("Failed to start consumer: %v", err)
		}
	}()
	// Create Notification Controller
	notificationController := controller.NewNotificationController(notificationService)

	// Create Gin router
	r := gin.Default()

	// Register routes
	router.RegisterNotificationRoutes(r, notificationController)

	// Register WebSocket route
	r.GET("/ws", func(c *gin.Context) {
		handlers.NewWebSocketHandler(notificationService, configurationService)(c.Writer, c.Request)
	})

	// Enable CORS for all origins
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-User-ID"},
		AllowCredentials: true,
	}).Handler(r)

	port := config.LoadConfig().Port
	log.Println("Server started on port: " + port)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}

func processAllowedOrigins() {
	origins := config.LoadConfig().AllowedOrigins
	if origins == "" {
		origins = "http://127.0.0.1:4200,http://localhost:4200"
	}
	allowedOrigins = strings.Split(origins, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
}
