package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/felipe/zemeow/internal/api/handlers"
	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/session"
)

// Server representa o servidor HTTP
type Server struct {
	app            *fiber.App
	config         *config.Config
	logger         logger.Logger
	sessionHandler *handlers.SessionHandler
	messageHandler *handlers.MessageHandler
	webhookHandler *handlers.WebhookHandler
	authMiddleware *middleware.AuthMiddleware
}

// NewServer cria uma nova instância do servidor
func NewServer(
	cfg *config.Config,
	sessionRepo repositories.SessionRepository,
	sessionService interface{},
	authService interface{},
) *Server {
	// Configurar Fiber
	app := fiber.New(fiber.Config{
		AppName:      "ZeMeow API",
		ServerHeader: "ZeMeow/1.0",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			return c.Status(code).JSON(fiber.Map{
				"error":     "INTERNAL_ERROR",
				"message":   err.Error(),
				"code":      code,
				"timestamp": time.Now().Unix(),
			})
		},
	})

	// Criar handlers
	sessionHandler := handlers.NewSessionHandler(sessionService.(session.Service))
	messageHandler := handlers.NewMessageHandler()
	webhookHandler := handlers.NewWebhookHandler()

	// Criar middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.AdminAPIKey, sessionRepo)

	return &Server{
		app:            app,
		config:         cfg,
		logger:         logger.GetWithSession("api_server"),
		sessionHandler: sessionHandler,
		messageHandler: messageHandler,
		webhookHandler: webhookHandler,
		authMiddleware: authMiddleware,
	}
}

// SetupRoutes configura todas as rotas da API
func (s *Server) SetupRoutes() {
	// Middleware global
	s.app.Use(recover.New())
	s.app.Use(s.authMiddleware.CORS())
	s.app.Use(s.authMiddleware.RequestLogger())

	// API routes (sem prefixo v1)
	api := s.app

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// Session routes (admin only - requer Admin API Key)
	sessions := api.Group("/sessions")
	sessions.Use(s.authMiddleware.RequireAdmin())
	sessions.Post("/", s.sessionHandler.CreateSession)
	sessions.Get("/", s.sessionHandler.GetAllSessions)
	sessions.Get("/active", s.sessionHandler.GetActiveConnections)

	// Session operations (requer Session API Key ou Admin API Key)
	sessions.Use(s.authMiddleware.RequireAuth())
	sessions.Get("/:sessionId", s.sessionHandler.GetSession)
	sessions.Put("/:sessionId", s.sessionHandler.UpdateSession)
	sessions.Delete("/:sessionId", s.sessionHandler.DeleteSession)
	sessions.Post("/:sessionId/connect", s.sessionHandler.ConnectSession)
	sessions.Post("/:sessionId/disconnect", s.sessionHandler.DisconnectSession)
	sessions.Post("/:sessionId/logout", s.sessionHandler.LogoutSession)
	sessions.Post("/:sessionId/pairphone", s.sessionHandler.PairPhone)
	sessions.Get("/:sessionId/status", s.sessionHandler.GetSessionStatus)
	sessions.Get("/:sessionId/qr", s.sessionHandler.GetSessionQRCode)
	sessions.Get("/:sessionId/stats", s.sessionHandler.GetSessionStats)

	// Proxy operations
	sessions.Post("/:sessionId/proxy", s.sessionHandler.SetProxy)
	sessions.Get("/:sessionId/proxy", s.sessionHandler.GetProxy)
	sessions.Post("/:sessionId/proxy/test", s.sessionHandler.TestProxy)

	// Message operations
	sessions.Post("/:sessionId/messages", s.messageHandler.SendMessage)
	sessions.Get("/:sessionId/messages", s.messageHandler.GetMessages)
	sessions.Post("/:sessionId/messages/bulk", s.messageHandler.SendBulkMessages)
	sessions.Get("/:sessionId/messages/:messageId/status", s.messageHandler.GetMessageStatus)

	// Webhook operations (admin only)
	webhooks := api.Group("/webhooks")
	webhooks.Use(s.authMiddleware.RequireAdmin())
	webhooks.Post("/send", s.webhookHandler.SendWebhook)
	webhooks.Get("/stats", s.webhookHandler.GetWebhookStats)
	webhooks.Post("/start", s.webhookHandler.StartWebhookService)
	webhooks.Post("/stop", s.webhookHandler.StopWebhookService)
	webhooks.Get("/status", s.webhookHandler.GetWebhookServiceStatus)
	webhooks.Get("/sessions/:sessionId/stats", s.webhookHandler.GetSessionWebhookStats)

	s.logger.Info().Msg("API routes configured successfully")
}

// Start inicia o servidor HTTP
func (s *Server) Start() error {
	// Configurar rotas
	s.SetupRoutes()

	// Obter porta da configuração
	port := s.config.Server.Port
	if port == 0 {
		port = 8080 // porta padrão
	}

	address := fmt.Sprintf(":%d", port)

	s.logger.Info().Int("port", port).Msg("Starting HTTP server")

	// Iniciar servidor
	return s.app.Listen(address)
}

// Stop para o servidor HTTP
func (s *Server) Stop() error {
	s.logger.Info().Msg("Stopping HTTP server")
	return s.app.Shutdown()
}

// GetApp retorna a instância do Fiber app
func (s *Server) GetApp() *fiber.App {
	return s.app
}
