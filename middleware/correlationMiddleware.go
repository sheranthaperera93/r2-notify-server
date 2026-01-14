package middleware

import (
	"r2-notify/utils"

	"github.com/gin-gonic/gin"
)

func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get correlation ID from header
		correlationID := c.Request.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = utils.GenerateUUID() // generate if missing
		}

		// Store in gin.Context
		c.Set("correlationId", correlationID)

		// Continue request
		c.Next()
	}
}
