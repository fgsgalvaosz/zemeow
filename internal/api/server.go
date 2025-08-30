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
	"github.com/felipe/zemeow/internal/service/auth"
	"github.com/felipe/zemeow/internal/service/meow"
	"github.com/felipe/zemeow/internal/service/webhook"
)

// Server representa o servidor HTTP
type Server struct {
	app             *fiber.App
	config          *config.Config
	logger          logger.Logger
	authHandler     *handlers.AuthHandler
	sessionHandler  *handlers.SessionHandler
	webhookHandler  *handlers.WebhookHandler
	authMiddleware  *middleware.AuthMiddleware
}

// NewServer cria uma nova instância do servidor
func NewServer(
	cfg *config.Config,
	sessionRepo repositories.SessionRepository,
	whatsappMgr *meow.WhatsAppManager,
	authService *auth.AuthService,
	webhookService *webhook.Service,
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
	authHandler := handlers.NewAuthHandler(authService)
	sessionHandler := handlers.NewSessionHandler(sessionRepo, whatsappMgr, authService)
	webhookHandler := handlers.NewWebhookHandler(webhookService, authService)

	// Criar middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	return &Server{
		app:            app,
		config:         cfg,
		logger:         logger.GetWithSession("api_server"),
		authHandler:    authHandler,
		sessionHandler: sessionHandler,
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

	// Health check
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// API v1 routes
	api := s.app.Group("/api/v1")

	// Auth routes (públicas)
	auth := api.Group("/auth")
	auth.Post("/login", s.authHandler.Login)
	auth.Post("/refresh", s.authHandler.RefreshToken)
	auth.Get("/validate", s.authHandler.ValidateToken)

	// Auth routes (autenticadas)
	authProtected := auth.Use(s.authMiddleware.RequireAuth())
	authProtected.Post("/token", s.authHandler.GenerateAPIToken)
	authProtected.Delete("/token", s.authHandler.RevokeToken)
	authProtected.Get("/info", s.authHandler.GetTokenInfo)

	// Session routes
	sessions := api.Group("/sessions")

	// Session routes (admin only)
	sessionsAdmin := sessions.Use(s.authMiddleware.RequireAdmin())
	sessionsAdmin.Post("/", s.sessionHandler.CreateSession)
	sessionsAdmin.Get("/", s.sessionHandler.GetAllSessions)
	sessionsAdmin.Get("/active", s.sessionHandler.GetActiveConnections)
	sessionsAdmin.Delete("/:sessionId", s.sessionHandler.DeleteSession)

	// Session routes (autenticadas com acesso à sessão)
	sessionsAuth := sessions.Use(s.authMiddleware.RequireAuth())
	sessionsAuth.Get("/:sessionId", s.sessionHandler.GetSession)
	sessionsAuth.Put("/:sessionId", s.sessionHandler.UpdateSession)
	sessionsAuth.Post("/:sessionId/connect", s.sessionHandler.ConnectSession)
	sessionsAuth.Post("/:sessionId/disconnect", s.sessionHandler.DisconnectSession)
	sessionsAuth.Get("/:sessionId/status", s.sessionHandler.GetSessionStatus)
	sessionsAuth.Get("/:sessionId/qr", s.sessionHandler.GetSessionQRCode)
	sessionsAuth.Get("/:sessionId/stats", s.sessionHandler.GetSessionStats)

	// Webhook routes
	webhooks := api.Group("/webhooks")

	// Webhook routes (admin only)
	webhooksAdmin := webhooks.Use(s.authMiddleware.RequireAdmin())
	webhooksAdmin.Post("/send", s.webhookHandler.SendWebhook)
	webhooksAdmin.Get("/stats", s.webhookHandler.GetWebhookStats)
	webhooksAdmin.Post("/start", s.webhookHandler.StartWebhookService)
	webhooksAdmin.Post("/stop", s.webhookHandler.StopWebhookService)
	webhooksAdmin.Get("/status", s.webhookHandler.GetWebhookServiceStatus)

	// Webhook routes (autenticadas com acesso à sessão)
	webhooksAuth := webhooks.Use(s.authMiddleware.RequireAuth())
	webhooksAuth.Get("/sessions/:sessionId/stats", s.webhookHandler.GetSessionWebhookStats)

	// Rate limiting para rotas públicas
	auth.Use(s.authMiddleware.RateLimit(60)) // 60 requests per minute

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