// @title ZeMeow WhatsApp API
// @version 1.0.0
// @description API para integração com WhatsApp usando a biblioteca whatsmeow. Permite criação, gerenciamento e comunicação através de sessões do WhatsApp, facilitando automações, envio de mensagens, gerenciamento de grupos e integração com webhooks.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:3000
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Bearer token ou API key para autenticação na API. Pode ser fornecido no header Authorization (Bearer <token>), X-API-Key ou apikey.
package main

import (
	"log"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/database"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/repositories"
	"github.com/felipe/zemeow/internal/server"
	"github.com/felipe/zemeow/internal/services/session"
	"github.com/felipe/zemeow/internal/services/webhook"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize logger
	logger.Init(cfg.Logging.Level, cfg.Logging.Pretty)

	// Initialize database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize repositories
	sessionRepo := repositories.NewSessionRepository(db.DB)
	// messageRepo será nil por enquanto

	// Initialize services
	sessionService := session.NewService(sessionRepo, nil) // redis será nil por enquanto
	webhookService := webhook.NewWebhookService(sessionRepo, cfg)

	// Create and start server
	apiServer := api.NewServer(
		cfg,
		sessionRepo,
		sessionService,
		nil, // authService pode ser nil por enquanto
		webhookService,
		nil, // messageRepo será nil por enquanto
	)

	log.Fatal(apiServer.Start())
}
