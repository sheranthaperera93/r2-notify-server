package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sheranthaperera93/r2-notify-server/internal/config"
	"github.com/sheranthaperera93/r2-notify-server/pkg/token"
)

const ContextUserID = "userID"
const ContextUserName = "username"
const ContextEmail = "email"

// RequireAuth validates the JWT Bearer token on protected HTTP routes (dashboard/key management).
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		cfg := config.LoadConfig()
		claims, err := token.ParseAccessToken(parts[1], cfg.JWTSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserName, claims.Username)
		c.Set(ContextEmail, claims.Email)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	id, _ := c.Get(ContextUserID)
	return id.(string)
}

func GetUserName(c *gin.Context) string {
	username, _ := c.Get(ContextUserName)
	return username.(string)
}
