package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"finance-dashboard/pkg/response"
)

func RoleGuard(allowed ...string) gin.HandlerFunc {
	set := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		set[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Error(c, http.StatusUnauthorized,
				"UNAUTHORIZED", "Authentication required")
			c.Abort()
			return
		}

		r, _ := role.(string)
		if _, ok := set[r]; !ok {
			response.Error(c, http.StatusForbidden,
				"INSUFFICIENT_PERMISSIONS",
				"You do not have permission to perform this action")
			c.Abort()
			return
		}
		c.Next()
	}
}
