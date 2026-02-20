package utils

import (
	"fmt"
	"r2-notify-server/data"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

func ValidateToken(tokenString string, jwtSecret []byte) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims.Subject, nil // Return subject (user ID) from claims
	}

	return "", fmt.Errorf("invalid token")
}
