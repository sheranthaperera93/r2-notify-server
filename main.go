package main

import (
	"context"
	"log"
	"net/http"
	"os"
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
	if os.Getenv("ENV") != data.PRODUCTION_ENV {
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
		origins = data.DEFAULT_ORIGINS
	}
	allowedOrigins = strings.Split(origins, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
}
