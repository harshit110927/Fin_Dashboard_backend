package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		requestID, _ := c.Get("request_id")
		log.Printf("[%s] [%s] %s | %d | %s | %s",
			requestID, c.Request.Method, c.Request.URL.Path,
			c.Writer.Status(), time.Since(start), c.ClientIP())
	}
}
