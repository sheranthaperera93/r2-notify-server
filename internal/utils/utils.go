package utils

import (
	"crypto/rand"
	"encoding/base64"
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

func GenerateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
