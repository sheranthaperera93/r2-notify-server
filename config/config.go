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
	mongoRetryWrites              string
	mongoSsl                      string
	RedisHost                     string
	RedisPort                     int
	RedisUsername                 string
	RedisPassword                 string
	RedisTLSEnabled               string
	EventHubNameSpaceConString    string
	EventHubNotificationEventName string
	AllowedOrigins                string
	LogLevel                      string
	LogMethod                     string
	LogFilePath                   string
	MaxLogFileSize                int
	AppInsightsInstrumentationKey string
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
		mongoRetryWrites:              GetEnv("MONGO_RETRY_WRITES", "true"),
		mongoSsl:                      GetEnv("MONGO_SSL", "false"),
		RedisHost:                     GetEnv("REDIS_HOST", "localhost"),
		RedisPort:                     GetEnvInt("REDIS_PORT", 6379),
		RedisUsername:                 GetEnv("REDIS_USERNAME", ""),
		RedisPassword:                 GetEnv("REDIS_PASSWORD", ""),
		RedisTLSEnabled:               GetEnv("REDIS_TLS_ENABLED", "false"),
		EventHubNameSpaceConString:    GetEnv("EVENT_HUB_NAMESPACE_CON_STRING", ""),
		EventHubNotificationEventName: GetEnv("EVENT_HUB_NOTIFICATION_EVENT_NAME", ""),
		AllowedOrigins:                GetEnv("ALLOWED_ORIGINS", "*"),
		LogLevel:                      GetEnv("LOG_LEVEL", ""),
		LogMethod:                     GetEnv("LOG_METHOD", "file"),
		LogFilePath:                   GetEnv("LOG_FILE_PATH", "./logs/app.log"),
		MaxLogFileSize:                GetEnvInt("MAX_LOG_FILE_SIZE", 10485760),
		AppInsightsInstrumentationKey: GetEnv("APP_INSIGHTS_INSTRUMENTATION_KEY", ""),
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
