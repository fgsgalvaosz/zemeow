package main

import (
	"context"
	"log"
	"os"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/database"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/repositories"
	"github.com/felipe/zemeow/internal/server"
	"github.com/felipe/zemeow/internal/services/session"
	"github.com/felipe/zemeow/internal/services/webhook"
	"github.com/joho/godotenv"
	"github.com/jmoiron/sqlx"
)

// @title ZeMeow WhatsApp API
// @version 1.0.0
// @description API para integração com WhatsApp usando a biblioteca whatsmeow. Permite criação, gerenciamento e comunicação através de sessões do WhatsApp, facilitando automações, envio de mensagens, gerenciamento de grupos e integração com webhooks.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Bearer token ou API key para autenticação na API. Pode ser fornecido no header Authorization (Bearer <token>), X-API-Key ou apikey.

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v\n", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger.InitWithConfig(cfg.Logging.Level, cfg.Logging.Pretty, cfg.Logging.Color, cfg.Logging.IncludeCaller)

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get SQL store for WhatsApp
	container := db.GetSQLStore()
	if container == nil {
		log.Fatalf("Failed to initialize WhatsApp SQL store")
	}

	// Run migrations if migrate command is provided
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := database.Migrate(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations completed successfully")
		return
	}

	// Initialize repositories
	sessionRepo := repositories.NewSessionRepository(db.DB)
	sqlxDb := sqlx.NewDb(db.DB, "postgres")
	messageRepo := repositories.NewMessageRepository(sqlxDb)

	// Initialize session manager with WhatsApp manager
	sessionManager := session.NewManager(container, sessionRepo, messageRepo, cfg)
	
	// Initialize services with session manager
	webhookService := webhook.NewWebhookService(sessionRepo, cfg)
	sessionService := session.NewService(sessionRepo, sessionManager) // Pass session manager to session service

	// Start webhook service
	webhookService.Start()
	defer webhookService.Stop()

	// Start session manager which will start WhatsApp manager
	if err := sessionManager.Start(); err != nil {
		log.Fatalf("Failed to start session manager: %v", err)
	}
	defer sessionManager.Shutdown(context.Background())

	// Create and start server
	server := api.NewServer(cfg, sessionRepo, sessionService, nil, webhookService, messageRepo)
	
	log.Printf("Starting ZeMeow API on %s:%d", cfg.Server.Host, cfg.Server.Port)
	
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}