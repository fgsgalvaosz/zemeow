// @title Zemeow WhatsApp API
// @version 1.0
// @description API para integração com WhatsApp usando whatsmeow
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

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
	dbConn, err := db.Connect(cfg)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	// Initialize database automatically
	logger.Get().Info().Msg("Initializing database...")

	// Run application migrations automatically
	if err := dbConn.Migrate(); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to run application migrations")
		os.Exit(1)
	}
	logger.Get().Info().Msg("Application migrations completed successfully")

	// Create application indexes
	if err := dbConn.CreateIndexes(); err != nil {
		logger.Get().Warn().Err(err).Msg("Failed to create application indexes")
		// Não vamos parar a aplicação por causa disso, apenas logar o warning
	} else {
		logger.Get().Info().Msg("Application indexes created successfully")
	}

	// Apply PostgreSQL optimizations
	if err := dbConn.OptimizeForWhatsApp(); err != nil {
		logger.Get().Warn().Err(err).Msg("Failed to apply PostgreSQL optimizations")
		// Não vamos parar a aplicação por causa disso, apenas logar o warning
	} else {
		logger.Get().Info().Msg("PostgreSQL optimizations applied successfully")
	}

	// Verify database setup
	if err := dbConn.VerifySetup(); err != nil {
		logger.Get().Warn().Err(err).Msg("Database setup verification failed")
		// Não vamos parar a aplicação, apenas logar o warning
	}

	logger.Get().Info().Msg("Database initialization completed")

	// Initialize WhatsApp SQL Store (this will create whatsmeow tables)
	logger.Get().Info().Msg("Initializing WhatsApp SQL Store...")
	sqlStore := dbConn.GetSQLStore()
	if sqlStore == nil {
		logger.Get().Error().Msg("Failed to initialize WhatsApp SQL Store")
		os.Exit(1)
	}
	logger.Get().Info().Msg("WhatsApp SQL Store initialized successfully")

	// Re-run migrations to ensure WhatsApp relationships are created
	logger.Get().Info().Msg("Ensuring WhatsApp relationships are created...")
	if err := dbConn.Migrate(); err != nil {
		logger.Get().Warn().Err(err).Msg("Failed to apply WhatsApp relationship migrations")
		// Não vamos parar a aplicação, apenas logar o warning
	}

	// Initialize repositories
	sessionRepo := repositories.NewSessionRepository(dbConn.DB)

	// Initialize services with WhatsApp store
	sessionService := session.NewService(sessionRepo, sqlStore)

	// Create server with WhatsApp store
	server := api.NewServer(cfg, sessionRepo, sessionService, sqlStore)

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
