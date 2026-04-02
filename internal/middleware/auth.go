package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"finance-dashboard/internal/config"
	jwtpkg "finance-dashboard/pkg/jwt"
	"finance-dashboard/pkg/response"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authorization header")
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtpkg.ValidateToken(tokenStr, config.C.JWTSecret)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
