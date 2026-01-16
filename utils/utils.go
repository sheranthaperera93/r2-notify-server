package utils

import (
	"r2-notify-server/data"
	"strings"

	"github.com/google/uuid"
)

func ProcessAllowedOrigins(origins string) []string {
	if origins == "" {
		origins = data.DEFAULT_ORIGINS
	}
	allowedOrigins := strings.Split(origins, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
	return allowedOrigins
}

func GenerateUUID() string {
	return uuid.New().String()
}
