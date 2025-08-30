package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/felipe/zemeow/internal/api"
	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/session"
)

func main() {
	// Initialize configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.Log)

	// Initialize database
	database, err := db.Initialize(cfg.Database)
	if err != nil {
		logger.Get().Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer database.Close()

	// Initialize repositories
	sessionRepo := repositories.NewSessionRepository(database)

	// Initialize services
	sessionService := session.NewService(sessionRepo)

	// Create server
	server := api.NewServer(cfg, sessionRepo, sessionService, nil)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Get().Error().Err(err).Msg("Server error")
			cancel()
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		logger.Get().Info().Msg("Shutting down server...")
	case <-ctx.Done():
		logger.Get().Info().Msg("Server context cancelled")
	}

	// Attempt graceful shutdown
	if err := server.Stop(); err != nil {
		logger.Get().Error().Err(err).Msg("Error stopping server")
	}

	logger.Get().Info().Msg("Server exited")
}