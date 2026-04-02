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
		role, _ := c.Get("role")
		r, _ := role.(string)
		if _, ok := set[r]; !ok {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
			return
		}
		c.Next()
	}
}
