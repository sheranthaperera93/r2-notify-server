package config

import (
	"os"
	"strconv"
)

type Config struct {
	Environment                   string
	Port                          string
	MongoHost                     string
	MongoPort                     int
	MongoDBName                   string
	MongoUserName                 string
	MongoPassword                 string
	RedisHost                     string
	RedisPort                     int
	RedisPassword                 string
	EventHubNameSpaceConString    string
	EventHubNotificationEventName string
	AllowedOrigins                string
}

func LoadConfig() *Config {
	return &Config{
		Environment:                   GetEnv("ENV", "development"),
		Port:                          GetEnv("PORT", "8081"),
		MongoHost:                     GetEnv("MONGO_HOST", "localhost"),
		MongoPort:                     GetEnvInt("MONGO_PORT", 27017),
		MongoDBName:                   GetEnv("MONGO_DB_NAME", "go_rampup"),
		MongoUserName:                 GetEnv("MONGO_USER_NAME", ""),
		MongoPassword:                 GetEnv("MONGO_PASSWORD", ""),
		RedisHost:                     GetEnv("REDIS_HOST", "localhost"),
		RedisPort:                     GetEnvInt("REDIS_PORT", 6379),
		RedisPassword:                 GetEnv("REDIS_PASSWORD", ""),
		EventHubNameSpaceConString:    GetEnv("EVENT_HUB_NAMESPACE_CON_STRING", ""),
		EventHubNotificationEventName: GetEnv("EVENT_HUB_NOTIFICATION_EVENT_NAME", ""),
		AllowedOrigins:                GetEnv("ALLOWED_ORIGINS", "*"),
	}
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
