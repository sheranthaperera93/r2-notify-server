package utils

import (
	"context"
	"fmt"
	"r2-notify-server/config"
	"r2-notify-server/data"
	"strings"

	// "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
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

func ValidateAPIKey(apiKey string) (string, error) {
	ctx := context.Background()

	unkeyClient := unkey.New(
		unkey.WithSecurity(config.LoadConfig().UnkeyRootKey),
	)

	res, err := unkeyClient.Keys.VerifyKey(ctx, components.V2KeysVerifyKeyRequestBody{
		Key: apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("API Key Request failed: %w", err)
	}

	body := res.V2KeysVerifyKeyResponseBody
	if body == nil {
		return "", fmt.Errorf("Empty response from API key verification")
	}

	if !body.Data.Valid {
		switch body.Data.Code {
		case "EXPIRED":
			return "", fmt.Errorf("API key has expired")
		case "DISABLED":
			return "", fmt.Errorf("API key is disabled")
		case "RATE_LIMITED":
			return "", fmt.Errorf("Rate limit exceeded")
		case "USAGE_EXCEEDED":
			return "", fmt.Errorf("Usage quota exceeded")
		default:
			return "", fmt.Errorf("invalid API key: %s", body.Data.Code)
		}
	}

	if body.Data.Identity == nil || body.Data.Identity.ExternalID == "" {
		return "", fmt.Errorf("API key has no identity set")
	}

	return body.Data.Identity.ExternalID, nil
}
