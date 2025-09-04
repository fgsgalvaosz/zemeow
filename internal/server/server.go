package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/handlers"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/middleware"
	"github.com/felipe/zemeow/internal/repositories"
	"github.com/felipe/zemeow/internal/routers"
	"github.com/felipe/zemeow/internal/services/session"
	"github.com/felipe/zemeow/internal/services/webhook"

	_ "github.com/felipe/zemeow/docs"
)

type Server struct {
	app    *fiber.App
	config *config.Config
	logger *logger.ComponentLogger
	router *routers.Router
}

func NewServer(
	cfg *config.Config,
	sessionRepo repositories.SessionRepository,
	sessionService session.Service,
	authService interface{},
	webhookService *webhook.WebhookService,
	messageRepo repositories.MessageRepository,
) *Server {

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

	// Create handlers
	sessionHandler := handlers.NewSessionHandler(sessionService, sessionRepo)
	webhookHandler := handlers.NewWebhookHandler(webhookService, sessionRepo)
	groupHandler := handlers.NewGroupHandler(sessionService)

	var messageHandler *handlers.MessageHandler
	if messageRepo != nil {
		messageHandler = handlers.NewMessageHandler(sessionService, nil) // mediaService nil por enquanto
	}

	var mediaHandler *handlers.MediaHandler
	// mediaHandler será nil por enquanto

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.AdminAPIKey, sessionRepo)

	// Create router
	routerConfig := &routers.RouterConfig{
		AuthMiddleware: authMiddleware,
		SessionHandler: sessionHandler,
		MessageHandler: messageHandler,
		WebhookHandler: webhookHandler,
		GroupHandler:   groupHandler,
		MediaHandler:   mediaHandler,
	}
	router := routers.NewRouter(app, routerConfig)

	return &Server{
		app:    app,
		config: cfg,
		logger: logger.ForComponent("api").WithSession("api_server"),
		router: router,
	}
}

func (s *Server) SetupRoutes() {
	setupOp := s.logger.ForOperation("setup_routes")

	s.app.Use(recover.New())
	s.router.SetupRoutes()

	setupOp.Success().
		Msg(logger.GetStandardizedMessage("api", "setup_routes", "success"))
}

func (s *Server) Start() error {
	s.SetupRoutes()

	startOp := s.logger.ForOperation("start_server")

	port := s.config.Server.Port
	if port == 0 {
		port = 8080
	}

	address := fmt.Sprintf(":%d", port)

	startOp.Starting().
		Int("port", port).
		Str("address", address).
		Msg(logger.GetStandardizedMessage("api", "start_server", "starting"))

	return s.app.Listen(address)
}

func (s *Server) Stop() error {
	stopOp := s.logger.ForOperation("stop_server")

	stopOp.Starting().
		Msg(logger.GetStandardizedMessage("api", "stop_server", "starting"))

	err := s.app.Shutdown()
	if err != nil {
		stopOp.Failed("SHUTDOWN_ERROR").
			Err(err).
			Msg(logger.GetStandardizedMessage("api", "stop_server", "failed"))
		return err
	}

	stopOp.Success().
		Msg(logger.GetStandardizedMessage("api", "stop_server", "success"))
	return nil
}

func (s *Server) GetApp() *fiber.App {
	return s.app
}