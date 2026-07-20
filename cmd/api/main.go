package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-minimal-backend/internal/0config"
	router "go-minimal-backend/internal/1delivery/http"
	"go-minimal-backend/internal/1delivery/http/handler"
	"go-minimal-backend/internal/2usecase"
	"go-minimal-backend/internal/3repository/postgres"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Setup PostgreSQL
	dbPool, err := postgres.New(cfg.DB_DSN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbPool.Close()

	// 3. Initialize Repositories
	userRepo := postgres.NewUserRepository(dbPool)
	noteRepo := postgres.NewNoteRepository(dbPool)

	// 4. Initialize Usecases
	tokenExpiration := 24 * time.Hour
	authUseCase := usecase.NewAuthUsecase(userRepo, cfg.JWT_SECRET, tokenExpiration)
	noteUseCase := usecase.NewNoteUsecase(noteRepo)

	// 5. Initialize Handlers
	authHandler := handler.NewAuthHandler(authUseCase)
	noteHandler := handler.NewNoteHandler(noteUseCase)

	// 6. Setup Router
	mux := router.NewRouter(authHandler, noteHandler, cfg.JWT_SECRET, cfg.CORS_ALLOWED_ORIGINS)

	// 7. Start Server
	server := &http.Server{
		Addr:    ":" + cfg.PORT,
		Handler: mux,
		// กัน slow client (เช่น slowloris) ที่ส่ง header/body ช้ามากๆ จนกิน connection ค้างไว้เรื่อยๆ
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// รัน server ใน goroutine แยก เพื่อให้ main goroutine ว่างไปรอ signal ปิดโปรแกรมต่อด้านล่าง
	go func() {
		log.Printf("Starting server on port %s", cfg.PORT)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// ดัก Ctrl+C (SIGINT) และ signal ที่ docker/k8s ส่งมาตอนสั่งปิด container (SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// ให้เวลา request ที่ค้างอยู่ตอนนี้ทำงานจนจบภายใน 10 วินาที ก่อนจะตัดจริง
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
