package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	Env            string
	AllowedOrigins string

	// MongoDB
	MongoSchema      string
	MongoHost        string
	MongoPort        int
	MongoDBName      string
	MongoUserName    string
	MongoPassword    string
	MongoRetryWrites bool
	MongoSsl         bool

	// Redis
	RedisHost       string
	RedisPort       int
	RedisUsername   string
	RedisPassword   string
	RedisTLSEnabled bool

	// Azure Event Hub
	EnableEventHub                bool
	EventHubNameSpaceConString    string
	EventHubNotificationEventName string

	// JWT (user dashboard sessions)
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	// Unkey (API key management + validation)
	UnkeyRootKey string
	UnkeyAPIID   string

	// Email (Resend)
	ResendAPIKey string
	ResendFrom   string

	AppBaseURL string

	// Logging
	LogLevel                      string
	LogMethod                     string
	LogFilePath                   string
	MaxLogFileSize                int
	AppInsightsInstrumentationKey string

	AllowedAPIKeyCount int
}

var loaded *Config

func LoadConfig() *Config {
	if loaded != nil {
		return loaded
	}

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	accessTTL, err := time.ParseDuration(GetEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		log.Fatalf("Invalid JWT_ACCESS_TTL: %v", err)
	}

	refreshTTL, err := time.ParseDuration(GetEnv("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		log.Fatalf("Invalid JWT_REFRESH_TTL: %v", err)
	}

	loaded = &Config{
		Port:           GetEnv("PORT", "8080"),
		Env:            GetEnv("ENV", "development"),
		AllowedOrigins: GetEnv("ALLOWED_ORIGINS", "*"),

		MongoSchema:      GetEnv("MONGO_SCHEMA", "mongodb"),
		MongoHost:        GetEnv("MONGO_HOST", "localhost"),
		MongoPort:        GetEnvInt("MONGO_PORT", 27017),
		MongoDBName:      GetEnv("MONGO_DB_NAME", ""),
		MongoUserName:    GetEnv("MONGO_USERNAME", ""),
		MongoPassword:    GetEnv("MONGO_PASSWORD", ""),
		MongoRetryWrites: GetEnvBool("MONGO_RETRY_WRITES", true),
		MongoSsl:         GetEnvBool("MONGO_SSL", false),

		RedisHost:       GetEnv("REDIS_HOST", "localhost"),
		RedisPort:       GetEnvInt("REDIS_PORT", 6379),
		RedisUsername:   GetEnv("REDIS_USERNAME", ""),
		RedisPassword:   GetEnv("REDIS_PASSWORD", ""),
		RedisTLSEnabled: GetEnvBool("REDIS_TLS_ENABLED", false),

		EnableEventHub:                GetEnvBool("ENABLE_EVENT_HUB", false),
		EventHubNameSpaceConString:    GetEnv("EVENT_HUB_NAMESPACE_CON_STRING", ""),
		EventHubNotificationEventName: GetEnv("EVENT_HUB_NOTIFICATION_EVENT_NAME", ""),

		JWTSecret:        GetEnv("JWT_SECRET", ""),
		JWTAccessExpiry:  accessTTL,
		JWTRefreshExpiry: refreshTTL,

		UnkeyRootKey: GetEnv("UNKEY_ROOT_KEY", ""),
		UnkeyAPIID:   GetEnv("UNKEY_API_ID", ""),

		ResendAPIKey: GetEnv("RESEND_API_KEY", ""),
		ResendFrom:   GetEnv("RESEND_FROM", "noreply@r2notify.dev"),
		AppBaseURL:   GetEnv("APP_BASE_URL", "http://localhost:5173"),

		LogLevel:                      GetEnv("LOG_LEVEL", "info"),
		LogMethod:                     GetEnv("LOG_METHOD", "file"),
		LogFilePath:                   GetEnv("LOG_FILE_PATH", "./logs/r2-notify.log"),
		MaxLogFileSize:                GetEnvInt("MAX_LOG_FILE_SIZE", 10),
		AppInsightsInstrumentationKey: GetEnv("APP_INSIGHTS_INSTRUMENTATION_KEY", ""),

		AllowedAPIKeyCount: GetEnvInt("ALLOWED_API_KEY_COUNT", 5),
	}

	return loaded
}

func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func GetEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func GetEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}
