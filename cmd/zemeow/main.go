package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felipe/zemeow/internal/api"
	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/meow"
	"github.com/felipe/zemeow/internal/service/webhook"
)

func main() {
	// Inicializar logger
	log := logger.GetWithSession("main")
	log.Info().Msg("Starting ZeMeow application")

	// Carregar configuração
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().Msg("Configuration loaded successfully")

	// Conectar ao banco de dados
	dbConn, err := db.Connect(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer dbConn.Close()

	log.Info().Msg("Database connected successfully")

	// Executar migrações
	if err := db.Migrate(dbConn); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	log.Info().Msg("Database migrations completed")

	// Inicializar repositórios
	sessionRepo := repositories.NewSessionRepository(dbConn)

	// Inicializar serviços
	whatsappMgr := meow.NewWhatsAppManager(sessionRepo, cfg)
	webhookService := webhook.NewWebhookService(cfg, whatsappMgr)

	log.Info().Msg("Services initialized successfully")

	// Inicializar servidor HTTP
	server := api.NewServer(
		cfg,
		sessionRepo,
		whatsappMgr,
		webhookService,
	)

	// Canal para capturar sinais do sistema
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor em goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	log.Info().Msg("ZeMeow application started successfully")

	// Aguardar sinal de parada
	<-sigChan
	log.Info().Msg("Shutdown signal received")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parar serviços
	log.Info().Msg("Stopping services...")

	// Parar webhook service
	webhookService.Stop()
	log.Info().Msg("Webhook service stopped")

	// Parar WhatsApp manager
	whatsappMgr.Stop()
	log.Info().Msg("WhatsApp manager stopped")

	// Parar servidor HTTP
	go func() {
		if err := server.Stop(); err != nil {
			log.Error().Err(err).Msg("Error stopping HTTP server")
		} else {
			log.Info().Msg("HTTP server stopped")
		}
	}()

	// Aguardar shutdown ou timeout
	select {
	case <-shutdownCtx.Done():
		log.Warn().Msg("Shutdown timeout reached")
	case <-time.After(5 * time.Second):
		log.Info().Msg("Graceful shutdown completed")
	}

	log.Info().Msg("ZeMeow application stopped")
}