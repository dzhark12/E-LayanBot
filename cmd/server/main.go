package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"elayan.local/desa/config"
	"elayan.local/desa/internal/app"
	"elayan.local/desa/internal/repository"
	"elayan.local/desa/internal/storage"
)

func main() {
	logger := log.New(os.Stdout, "elayandesa ", log.LstdFlags|log.LUTC)
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("configuration error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := repository.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database connection failed; check DATABASE_URL and database availability")
	}
	defer db.Close()
	if err := repository.Migrate(ctx, db); err != nil {
		logger.Fatalf("database migration failed: %v", err)
	}
	if err := repository.Seed(ctx, db, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		logger.Fatalf("initial data setup failed: %v", err)
	}
	files, err := storage.NewLocal(cfg.StorageDir)
	if err != nil {
		logger.Fatalf("private storage setup failed: %v", err)
	}
	application, err := app.New(db, cfg, files)
	if err != nil {
		logger.Fatalf("application setup failed: %v", err)
	}
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           application.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serviceCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go application.RunRetention(serviceCtx, logger)
	go func() {
		<-serviceCtx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Printf("dashboard and API listening on %s", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("http server failed: %v", err)
	}
}
