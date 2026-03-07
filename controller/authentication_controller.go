package controller

import (
	"net/http"
	"r2-notify-server/data"
	authenticationService "r2-notify-server/services/authentication"

	"github.com/gin-gonic/gin"
)

type AuthenticationController struct {
	authenticationService authenticationService.AuthenticationService
}

func NewAuthController(service authenticationService.AuthenticationService) *AuthenticationController {
	return &AuthenticationController{authenticationService: service}
}

func (controller *AuthenticationController) GoogleAuthHandler(ctx *gin.Context) {
	var req data.GoogleAuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, jwt, err := controller.authenticationService.GoogleAuthenticate(req.Token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"jwt":  jwt,
		"user": user,
	})
}
