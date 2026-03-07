package authenticationService

import (
	"context"
	"fmt"
	"r2-notify-server/config"
	"r2-notify-server/data"
	"r2-notify-server/logger"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/api/idtoken"
)

type AuthenticationServiceImpl struct {
	context context.Context
}

func NewAuthenticationServiceImpl() (service AuthenticationService, err error) {
	return &AuthenticationServiceImpl{
		context: context.Background(),
	}, err
}

func (t AuthenticationServiceImpl) GoogleAuthenticate(token string) (data.UserInfo, string, error) {

	clientID := config.LoadConfig().GoogleClientId
	payload, err := idtoken.Validate(t.context, token, clientID)
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Authentication Service",
			Operation: "GoogleAuthenticate",
			Message:   "Invalid Google token",
			Error:     err,
			UserId:    "",
		})
		return data.UserInfo{}, "", fmt.Errorf("Invalid Google token")
	}

	user := data.UserInfo{
		ID:     payload.Subject,
		Name:   payload.Claims["name"].(string),
		Email:  payload.Claims["email"].(string),
		Avatar: payload.Claims["picture"].(string),
	}

	// Issue JWT
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := jwtToken.SignedString([]byte(config.LoadConfig().JwtSecret))
	if err != nil {
		logger.Log.Error(logger.LogPayload{
			Component: "Authentication Service",
			Operation: "GoogleAuthenticate",
			Message:   "Token generation failed",
			Error:     err,
			UserId:    "",
		})
		return data.UserInfo{}, "", fmt.Errorf("Token generation failed")
	}
	logger.Log.Info(logger.LogPayload{
		Component: "Authentication Service",
		Operation: "GoogleAuthenticate",
		Message:   "Successfully authenticated user with Google",
		UserId:    user.ID,
	})
	return user, signed, nil
}
