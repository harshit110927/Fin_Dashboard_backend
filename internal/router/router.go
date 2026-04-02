package router

import (
	"github.com/gin-gonic/gin"

	"finance-dashboard/internal/handler"
	"finance-dashboard/internal/middleware"
)

func SetupRouter(authH *handler.AuthHandler, userH *handler.UserHandler, recH *handler.RecordHandler, dashH *handler.DashboardHandler, categoryH *handler.CategoryHandler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestLogger())
	r.Use(middleware.RateLimiter())

	v1 := r.Group("/api/v1")

	// Auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.Refresh)
		auth.POST("/logout", middleware.AuthRequired(), authH.Logout)
	}

	// User routes
	users := v1.Group("/users", middleware.AuthRequired(), middleware.RoleGuard("admin"))
	{
		users.GET("/", userH.List)
		users.GET("/:id", userH.GetByID)
		users.POST("/", userH.Create)
		users.PATCH("/:id", userH.Update)
		users.PATCH("/:id/role", userH.UpdateRole)
		users.PATCH("/:id/status", userH.UpdateStatus)
		users.DELETE("/:id", userH.Delete)
	}

	// Record routes
	records := v1.Group("/records", middleware.AuthRequired())
	{
		records.GET("/", middleware.RoleGuard("viewer", "analyst", "admin"), recH.List)
		records.GET("/:id", middleware.RoleGuard("viewer", "analyst", "admin"), recH.GetByID)
		records.POST("/", middleware.RoleGuard("admin"), recH.Create)
		records.PATCH("/:id", middleware.RoleGuard("admin"), recH.Update)
		records.DELETE("/:id", middleware.RoleGuard("admin"), recH.Delete)
		records.POST("/:id/void", middleware.RoleGuard("admin"), recH.Void)
	}

	// Dashboard routes
	dashboard := v1.Group("/dashboard", middleware.AuthRequired())
	{
		dashboard.GET("/summary", middleware.RoleGuard("viewer", "analyst", "admin"), dashH.Summary)
		dashboard.GET("/trends", middleware.RoleGuard("analyst", "admin"), dashH.Trends)
		dashboard.GET("/categories", middleware.RoleGuard("viewer", "analyst", "admin"), dashH.Categories)
		dashboard.GET("/recent", middleware.RoleGuard("viewer", "analyst", "admin"), dashH.Recent)
	}

	// Category routes
	categories := v1.Group("/categories", middleware.AuthRequired())
	{
		categories.GET("/", middleware.RoleGuard("viewer", "analyst", "admin"), categoryH.List)
		categories.POST("/", middleware.RoleGuard("admin"), categoryH.Create)
		categories.PATCH("/:id", middleware.RoleGuard("admin"), categoryH.Update)
	}

	return r
}
