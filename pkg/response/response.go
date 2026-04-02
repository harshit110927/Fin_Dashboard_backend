package response

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, map[string]interface{}{"success": true, "data": data})
}

func SuccessPaginated(c *gin.Context, status int, data interface{}, page, perPage, total int) {
	c.JSON(status, map[string]interface{}{
		"success": true,
		"data":    data,
		"meta":    map[string]int{"page": page, "per_page": perPage, "total": total},
	})
}

func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, map[string]interface{}{
		"success": false,
		"error":   map[string]string{"code": code, "message": message},
	})
}
