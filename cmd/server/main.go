package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	recH := handler.NewRecordHandler(recordSvc, auditRepo)
	dashH := handler.NewDashboardHandler(dashSvc)
	categoryH := handler.NewCategoryHandler(dashSvc)

	// 6. Start server
	r := router.SetupRouter(database, authH, userH, recH, dashH, categoryH)
	server := &http.Server{Addr: ":" + config.C.ServerPort, Handler: r}

	go func() {
		log.Printf("server starting on :%s", config.C.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown ensures in-flight requests complete before the process
	// exits. In financial systems this is critical — a mid-flight record create
	// followed by an audit log write must not be interrupted mid-transaction.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
}
