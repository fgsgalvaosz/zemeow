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

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}


	logger.Init(cfg.Logging.Level, cfg.Logging.Pretty)


	dbConn, err := db.Connect(cfg)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbConn.Close()


	logger.Get().Info().Msg("Initializing database...")


	if err := dbConn.Migrate(); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to run application migrations")
		os.Exit(1)
	}
	logger.Get().Info().Msg("Application migrations completed successfully")


	if err := dbConn.CreateIndexes(); err != nil {
		logger.Get().Warn().Err(err).Msg("Failed to create application indexes")

	} else {
		logger.Get().Info().Msg("Application indexes created successfully")
	}


	if err := dbConn.OptimizeForWhatsApp(); err != nil {
		logger.Get().Warn().Err(err).Msg("Failed to apply PostgreSQL optimizations")

	} else {
		logger.Get().Info().Msg("PostgreSQL optimizations applied successfully")
	}


	if err := dbConn.VerifySetup(); err != nil {
		logger.Get().Warn().Err(err).Msg("Database setup verification failed")

	}

	logger.Get().Info().Msg("Database initialization completed")


	logger.Get().Info().Msg("Initializing WhatsApp SQL Store...")
	sqlStore := dbConn.GetSQLStore()
	if sqlStore == nil {
		logger.Get().Error().Msg("Failed to initialize WhatsApp SQL Store")
		os.Exit(1)
	}
	logger.Get().Info().Msg("WhatsApp SQL Store initialized successfully")


	logger.Get().Info().Msg("Ensuring WhatsApp relationships are created...")
	if err := dbConn.Migrate(); err != nil {
		logger.Get().Warn().Err(err).Msg("Failed to apply WhatsApp relationship migrations")

	}


	sessionRepo := repositories.NewSessionRepository(dbConn.DB)


	sessionManager := session.NewManager(sqlStore, sessionRepo, cfg)


	if err := sessionManager.Start(); err != nil {
		logger.Get().Error().Err(err).Msg("Failed to start session manager")
		os.Exit(1)
	}


	sessionService := session.NewService(sessionRepo, sessionManager)


	server := api.NewServer(cfg, sessionRepo, sessionService, sqlStore)


	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	go func() {
		if err := server.Start(); err != nil {
			logger.Get().Error().Err(err).Msg("Server error")
			cancel()
		}
	}()


	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		logger.Get().Info().Msg("Shutting down server...")
	case <-ctx.Done():
		logger.Get().Info().Msg("Server context cancelled")
	}


	if err := server.Stop(); err != nil {
		logger.Get().Error().Err(err).Msg("Error stopping server")
	}


	if err := sessionManager.Shutdown(ctx); err != nil {
		logger.Get().Error().Err(err).Msg("Error shutting down session manager")
	}

	logger.Get().Info().Msg("Server exited")
}
