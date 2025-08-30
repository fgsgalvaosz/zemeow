package main

import (
	"log"
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
	// Carregar configurações
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validar configurações
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Inicializar logger
	logger.Init(cfg.Logging.Level, cfg.Logging.Pretty)
	mainLogger := logger.GetWithSession("main")

	mainLogger.Info().Str("environment", cfg.Server.Environment).Msg("Starting ZeMeow API server")

	// Conectar ao banco de dados
	database, err := db.New(&cfg.Database)
	if err != nil {
		mainLogger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	// Executar migrations
	mainLogger.Info().Msg("Running database migrations")
	if err := database.Migrate(); err != nil {
		mainLogger.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// Aplicar otimizações para WhatsApp
	if err := database.OptimizeForWhatsApp(); err != nil {
		mainLogger.Warn().Err(err).Msg("Failed to apply WhatsApp optimizations")
	}

	// Criar índices otimizados
	if err := database.CreateIndexes(); err != nil {
		mainLogger.Warn().Err(err).Msg("Failed to create optimized indexes")
	}

	// Inicializar repositórios
	sessionRepo := repositories.NewSessionRepository(database.DB)

	// Inicializar WhatsApp Manager
	whatsAppManager := meow.NewWhatsAppManager(database.DB, sessionRepo, cfg)
	if err := whatsAppManager.Start(); err != nil {
		mainLogger.Fatal().Err(err).Msg("Failed to start WhatsApp manager")
	}
	defer whatsAppManager.Stop()

	// Inicializar serviços
	sessionService := session.NewService(sessionRepo, whatsAppManager)

	// Criar servidor HTTP
	server := api.NewServer(cfg, sessionRepo, sessionService, nil)

	// Canal para capturar sinais do sistema
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor em goroutine
	go func() {
		mainLogger.Info().Int("port", cfg.Server.Port).Msg("HTTP server listening")
		if err := server.Start(); err != nil {
			mainLogger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// Aguardar sinal de parada
	<-quit
	mainLogger.Info().Msg("Shutting down server...")

	// Parar servidor graciosamente
	if err := server.Stop(); err != nil {
		mainLogger.Error().Err(err).Msg("Failed to stop server gracefully")
	}

	mainLogger.Info().Msg("ZeMeow API server stopped")
}