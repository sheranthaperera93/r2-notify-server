package utils

import (
	"r2-notify/data"
	"strings"
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
