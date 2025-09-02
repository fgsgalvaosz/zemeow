package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"

	"github.com/felipe/zemeow/internal/handlers"
	"github.com/felipe/zemeow/internal/middleware"

	_ "github.com/felipe/zemeow/docs"
)

type Router struct {
	app                  *fiber.App
	authMiddleware       *middleware.AuthMiddleware
	validationMiddleware *middleware.ValidationMiddleware
	sessionHandler       *handlers.SessionHandler
	messageHandler       *handlers.MessageHandler
	webhookHandler       *handlers.WebhookHandler
	groupHandler         *handlers.GroupHandler
	mediaHandler         *handlers.MediaHandler
}

type RouterConfig struct {
	AuthMiddleware *middleware.AuthMiddleware
	SessionHandler *handlers.SessionHandler
	MessageHandler *handlers.MessageHandler
	WebhookHandler *handlers.WebhookHandler
	GroupHandler   *handlers.GroupHandler
	MediaHandler   *handlers.MediaHandler
}

func NewRouter(app *fiber.App, config *RouterConfig) *Router {
	return &Router{
		app:                  app,
		authMiddleware:       config.AuthMiddleware,
		validationMiddleware: middleware.NewValidationMiddleware(),
		sessionHandler:       config.SessionHandler,
		messageHandler:       config.MessageHandler,
		webhookHandler:       config.WebhookHandler,
		groupHandler:         config.GroupHandler,
		mediaHandler:         config.MediaHandler,
	}
}

func (r *Router) SetupRoutes() {

	r.setupGlobalMiddleware()

	r.setupSwaggerRoutes()

	r.setupWebRoutes()

	r.setupHealthRoutes()

	r.setupSessionRoutes()

	r.setupWebhookRoutes()
}

func (r *Router) setupSwaggerRoutes() {

	config := swagger.Config{
		URL:          "doc.json",
		DeepLinking:  false,
		DocExpansion: "list",
	}

	r.app.Get("/swagger/*", swagger.New(config))

	r.app.Get("/docs/*", swagger.New(config))

	r.app.Get("/swagger", func(c *fiber.Ctx) error {
		return c.Redirect("/swagger/index.html", 301)
	})
}

func (r *Router) setupWebRoutes() {
	// Serve static files from web directory with proper routing
	r.app.Static("/web", "./web", fiber.Static{
		Browse: false,
		Index:  "index.html",
	})

	// Redirect /web to /web/
	r.app.Get("/web", func(c *fiber.Ctx) error {
		return c.Redirect("/web/", 301)
	})
}
