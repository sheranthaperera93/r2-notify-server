package middleware

import (
	"r2-notify/logger"
	"r2-notify/utils"

	"github.com/gin-gonic/gin"
)

func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get correlation ID from header
		correlationID := c.Request.Header.Get("X-Correlation-ID")
		logger.Log.Info(logger.LogPayload{
			Service:       "CorrelationMiddleware",
			Component:     "Middleware",
			Operation:     "CorrelationIDMiddleware",
			Message:       "Extracting X-Correlation-ID from request header",
			UserId:        c.Request.Header.Get("X-User-ID"),
			AppId:         c.Request.Header.Get("X-App-ID"),
			CorrelationId: correlationID,
		})
		if correlationID == "" {
			correlationID = utils.GenerateUUID()
			logger.Log.Info(logger.LogPayload{
				Service:       "CorrelationMiddleware",
				Component:     "Middleware",
				Operation:     "CorrelationIDMiddleware",
				Message:       "X-Correlation-ID is missing, generated new correlation ID",
				UserId:        c.Request.Header.Get("X-User-ID"),
				AppId:         c.Request.Header.Get("X-App-ID"),
				CorrelationId: correlationID,
			})
		}

		// Store in gin.Context
		c.Set("correlationId", correlationID)

		// Continue request
		c.Next()
	}
}
