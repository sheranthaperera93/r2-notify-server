package utils

import (
	"strings"

	"github.com/google/uuid"
	"github.com/sheranthaperera93/r2-notify-server/internal/data"
)

func ProcessAllowedOrigins(origins string) []string {
	if origins == "*" {
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
