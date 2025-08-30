package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/felipe/zemeow/internal/api/handlers"
	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/api/routes"
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
	router         *routes.Router
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
	sessionHandler := handlers.NewSessionHandler(sessionService.(session.Service), sessionRepo)
	messageHandler := handlers.NewMessageHandler(sessionService.(session.Service))
	webhookHandler := handlers.NewWebhookHandler()
	groupHandler := handlers.NewGroupHandler(sessionService.(session.Service))

	// Criar middleware de autenticação
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.AdminAPIKey, sessionRepo)

	// Configurar router
	routerConfig := &routes.RouterConfig{
		AuthMiddleware: authMiddleware,
		SessionHandler: sessionHandler,
		MessageHandler: messageHandler,
		WebhookHandler: webhookHandler,
		GroupHandler:   groupHandler,
	}
	router := routes.NewRouter(app, routerConfig)

	return &Server{
		app:    app,
		config: cfg,
		logger: logger.GetWithSession("api_server"),
		router: router,
	}
}

// SetupRoutes configura todas as rotas da API
func (s *Server) SetupRoutes() {
	// Middleware global de recuperação de pânico
	s.app.Use(recover.New())

	// Configurar rotas usando o router modular
	s.router.SetupRoutes()

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
