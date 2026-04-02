package main

import (
	"log"

	"finance-dashboard/internal/config"
	"finance-dashboard/internal/db"
	"finance-dashboard/internal/handler"
	"finance-dashboard/internal/repository"
	"finance-dashboard/internal/router"
	"finance-dashboard/internal/service"
)

func main() {
	// 1. Load config
	if err := config.Load(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	// 2. Connect to database
	database, err := db.Connect(config.C)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}

	// 3. Repositories
	userRepo := repository.NewUserRepository(database)
	tokenRepo := repository.NewTokenRepository(database)
	recordRepo := repository.NewRecordRepository(database)
	auditRepo := repository.NewAuditRepository(database)
	dashRepo := repository.NewDashboardRepository(database)

	// 4. Services
	authSvc := service.NewAuthService(userRepo, tokenRepo)
	recordSvc := service.NewRecordService(recordRepo, auditRepo)
	dashSvc := service.NewDashboardService(dashRepo)

	// 5. Handlers
	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(userRepo, auditRepo)
	recH := handler.NewRecordHandler(recordSvc)
	dashH := handler.NewDashboardHandler(dashSvc)
	categoryH := handler.NewCategoryHandler(dashSvc)

	// 6. Start server
	r := router.SetupRouter(authH, userH, recH, dashH, categoryH)
	log.Printf("server starting on :%s", config.C.ServerPort)
	if err := r.Run(":" + config.C.ServerPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
