package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/api/handlers"
	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/api/validators"
)

// Router estrutura principal do roteador
type Router struct {
	app                *fiber.App
	authMiddleware     *middleware.AuthMiddleware
	validationMiddleware *validators.ValidationMiddleware
	sessionHandler     *handlers.SessionHandler
	messageHandler     *handlers.MessageHandler
	webhookHandler     *handlers.WebhookHandler
}

// RouterConfig configuração do router
type RouterConfig struct {
	AuthMiddleware     *middleware.AuthMiddleware
	SessionHandler     *handlers.SessionHandler
	MessageHandler     *handlers.MessageHandler
	WebhookHandler     *handlers.WebhookHandler
}

// NewRouter cria uma nova instância do router
func NewRouter(app *fiber.App, config *RouterConfig) *Router {
	return &Router{
		app:                app,
		authMiddleware:     config.AuthMiddleware,
		validationMiddleware: validators.NewValidationMiddleware(),
		sessionHandler:     config.SessionHandler,
		messageHandler:     config.MessageHandler,
		webhookHandler:     config.WebhookHandler,
	}
}

// SetupRoutes configura todas as rotas da aplicação
func (r *Router) SetupRoutes() {
	// Middleware global
	r.setupGlobalMiddleware()
	
	// Health check
	r.setupHealthRoutes()
	
	// Rotas de sessão (incluindo mensagens)
	r.setupSessionRoutes()
	
	// Rotas de webhook
	r.setupWebhookRoutes()
}