package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/felipe/zemeow/internal/api/handlers"
	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/api/routes"
	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/media"
	"github.com/felipe/zemeow/internal/service/session"
	"github.com/felipe/zemeow/internal/service/webhook"

	"github.com/felipe/zemeow/docs"
)

type Server struct {
	app    *fiber.App
	config *config.Config
	logger logger.Logger
	router *routes.Router
}

func NewServer(
	cfg *config.Config,
	sessionRepo repositories.SessionRepository,
	sessionService interface{},
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

	configureSwagger(cfg)

	var mediaService *media.MediaService
	if cfg.MinIO.Endpoint != "" {
		var err error
		mediaService, err = media.NewMediaServiceFromConfig(&cfg.MinIO)
		if err != nil {
			logger.Get().Warn().Err(err).Msg("Failed to initialize MediaService, media routes will be disabled")
			mediaService = nil
		}
	}

	sessionHandler := handlers.NewSessionHandler(sessionService.(session.Service), sessionRepo)
	messageHandler := handlers.NewMessageHandler(sessionService.(session.Service), mediaService)
	webhookHandler := handlers.NewWebhookHandler(webhookService, sessionRepo)
	groupHandler := handlers.NewGroupHandler(sessionService.(session.Service))

	var mediaHandler *handlers.MediaHandler
	if mediaService != nil {
		mediaHandler = handlers.NewMediaHandler(mediaService, messageRepo)
	}

	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.AdminAPIKey, sessionRepo)

	routerConfig := &routes.RouterConfig{
		AuthMiddleware: authMiddleware,
		SessionHandler: sessionHandler,
		MessageHandler: messageHandler,
		WebhookHandler: webhookHandler,
		GroupHandler:   groupHandler,
		MediaHandler:   mediaHandler,
	}
	router := routes.NewRouter(app, routerConfig)

	return &Server{
		app:    app,
		config: cfg,
		logger: logger.GetWithSession("api_server"),
		router: router,
	}
}

func (s *Server) SetupRoutes() {

	s.app.Use(recover.New())

	s.router.SetupRoutes()

	s.logger.Info().Msg("API routes configured successfully")
}

func (s *Server) Start() error {

	s.SetupRoutes()

	port := s.config.Server.Port
	if port == 0 {
		port = 8080
	}

	address := fmt.Sprintf(":%d", port)

	s.logger.Info().Int("port", port).Msg("Starting HTTP server")

	return s.app.Listen(address)
}

func (s *Server) Stop() error {
	s.logger.Info().Msg("Stopping HTTP server")
	return s.app.Shutdown()
}

func (s *Server) GetApp() *fiber.App {
	return s.app
}

func configureSwagger(cfg *config.Config) {
	// Get the configured host from environment or config
	host := cfg.Server.Host

	// Check if we have a custom domain configured via environment
	customDomain := os.Getenv("SERVER_HOST")
	if customDomain == "0.0.0.0" || customDomain == "" {
		customDomain = ""
	}

	// Determine the actual host for Swagger
	var swaggerHost string
	var scheme string

	if customDomain != "" && customDomain != "localhost" {
		// Use custom domain (production)
		swaggerHost = customDomain
		scheme = "https"
		if strings.Contains(customDomain, "localhost") || strings.Contains(customDomain, "127.0.0.1") {
			scheme = "http"
		}
	} else if cfg.IsProduction() {
		// Production without custom domain - try to detect from config
		if strings.Contains(host, ".com") || strings.Contains(host, ".br") || strings.Contains(host, ".net") || strings.Contains(host, ".org") {
			swaggerHost = host
			scheme = "https"
		} else {
			// Fallback for production - use a placeholder
			swaggerHost = "api.yourdomain.com"
			scheme = "https"
		}
	} else {
		// Development mode
		if host == "0.0.0.0" || host == "" {
			host = "localhost"
		}
		swaggerHost = fmt.Sprintf("%s:%d", host, cfg.Server.Port)
		scheme = "http"
	}

	// Set both schemes to support both HTTP and HTTPS
	schemes := []string{scheme}
	if scheme == "https" {
		schemes = []string{"https", "http"} // Prefer HTTPS but allow HTTP
	} else {
		schemes = []string{"http", "https"} // Prefer HTTP but allow HTTPS
	}

	docs.SwaggerInfo.Host = swaggerHost
	docs.SwaggerInfo.Schemes = schemes

	logger.Get().Info().
		Str("swagger_host", swaggerHost).
		Strs("swagger_schemes", schemes).
		Str("config_host", cfg.Server.Host).
		Str("env_server_host", os.Getenv("SERVER_HOST")).
		Bool("is_production", cfg.IsProduction()).
		Msg("Swagger configured dynamically")
}
