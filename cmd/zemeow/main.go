package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/felipe/zemeow/internal/api"
	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/meow"
	"github.com/felipe/zemeow/internal/service/session"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.Logging.Level, cfg.Logging.Pretty)

	// Initialize database
	dbConn, err := db.New(&cfg.Database)
	if err != nil {
		logger.Get().Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer dbConn.Close()

	database := dbConn.DB

	// Initialize repositories
	sessionRepo := repositories.NewSessionRepository(database)

	// Initialize WhatsApp manager
	whatsappManager := meow.NewWhatsAppManager(database, sessionRepo, cfg)
	if err := whatsappManager.Start(); err != nil {
		logger.Get().Fatal().Err(err).Msg("Failed to start WhatsApp manager")
	}

	// Initialize session manager
	sessionManager := session.NewManager(nil, sessionRepo, cfg)
	if err := sessionManager.Start(); err != nil {
		logger.Get().Fatal().Err(err).Msg("Failed to start session manager")
	}

	// Initialize services
	sessionService := session.NewService(sessionRepo, sessionManager)

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