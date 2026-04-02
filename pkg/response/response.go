package response

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

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

func ValidationError(c *gin.Context, err error) {
	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		Error(c, 422, "VALIDATION_ERROR", err.Error())
		return
	}

	fields := make([]map[string]string, 0, len(verrs))
	for _, fe := range verrs {
		fields = append(fields, map[string]string{
			"field":   strings.ToLower(fe.Field()),
			"tag":     fe.Tag(),
			"message": fe.Error(),
		})
	}

	c.AbortWithStatusJSON(422, map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    "VALIDATION_ERROR",
			"message": "request validation failed",
			"fields":  fields,
		},
	})
}
