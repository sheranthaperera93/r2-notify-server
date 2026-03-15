package controller

import (
	"net/http"
	authenticationService "r2-notify-server/services/authentication"

	"github.com/gin-gonic/gin"
)

type AuthenticationController struct {
	authenticationService authenticationService.AuthenticationService
}

func NewAuthController(service authenticationService.AuthenticationService) *AuthenticationController {
	return &AuthenticationController{authenticationService: service}
}

func (controller *AuthenticationController) ApiKeyAuthHandler(ctx *gin.Context) {
	// This handler is intentionally left blank as API key authentication is handled in the middleware.
	ctx.JSON(http.StatusOK, gin.H{"message": "API key is valid"})
}
