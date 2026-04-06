package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// HealthCheck returns a simple liveness probe response including DB reachability.
func HealthCheck(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "ok"
		if err := db.PingContext(c.Request.Context()); err != nil {
			dbStatus = "unreachable"
		}
		requestID, _ := c.Get("request_id")
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"db":         dbStatus,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"request_id": requestID,
		})
	}
}
