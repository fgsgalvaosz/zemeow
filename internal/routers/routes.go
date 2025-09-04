package routers

import (
	"github.com/felipe/zemeow/internal/dto"
	"github.com/felipe/zemeow/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

func (r *Router) setupGlobalMiddleware() {
	r.app.Use(r.authMiddleware.CORSMiddleware())

	loggingMiddleware := middleware.NewLoggingMiddleware()
	r.app.Use(loggingMiddleware.Logger())
}

func (r *Router) setupHealthRoutes() {
	r.app.Get("/health", r.healthCheck)
}

// @Summary Health Check
// @Description Verifica o status da API
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "API funcionando corretamente"
// @Router /health [get]
func (r *Router) healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"service":   "zemeow-api",
		"version":   "1.0.0",
		"timestamp": "1640995200",
	})
}

func (r *Router) setupSessionRoutes() {

	sessions := r.app.Group("/sessions")

	r.setupGlobalSessionRoutes(sessions)

	r.setupSessionOperationRoutes(sessions)

	r.setupMessageRoutes()

	r.setupMediaRoutes()
	
	r.setupWebhookRoutes()
}

func (r *Router) setupGlobalSessionRoutes(sessions fiber.Router) {
	globalRoutes := sessions.Group("/", r.authMiddleware.RequireGlobalAPIKey())

	globalRoutes.Post("/add",
		r.validationMiddleware.ValidateJSON(&dto.CreateSessionRequest{}),
		r.sessionHandler.CreateSession,
	)

	globalRoutes.Get("/",
		r.validationMiddleware.ValidatePaginationParams(),
		r.sessionHandler.GetAllSessions,
	)

	globalRoutes.Get("/active", r.sessionHandler.GetActiveConnections)
}

func (r *Router) setupSessionOperationRoutes(sessions fiber.Router) {

	sessionRoutes := sessions.Group("/",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
	)

	sessionRoutes.Get("/:sessionId",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetSession,
	)

	sessionRoutes.Put("/:sessionId",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.UpdateSessionRequest{}),
		r.sessionHandler.UpdateSession,
	)

	sessionRoutes.Delete("/:sessionId",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.DeleteSession,
	)

	sessionRoutes.Post("/:sessionId/connect",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.ConnectSession,
	)

	sessionRoutes.Post("/:sessionId/disconnect",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.DisconnectSession,
	)

	sessionRoutes.Post("/:sessionId/logout",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.LogoutSession,
	)

	sessionRoutes.Get("/:sessionId/status",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetSessionStatus,
	)

	sessionRoutes.Get("/:sessionId/qr",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetSessionQRCode,
	)

	sessionRoutes.Get("/:sessionId/stats",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetSessionStats,
	)

	sessionRoutes.Post("/:sessionId/pairphone",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.PairPhoneRequest{}),
		r.sessionHandler.PairPhone,
	)

	sessionRoutes.Post("/:sessionId/proxy",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ProxyRequest{}),
		r.sessionHandler.SetProxy,
	)

	sessionRoutes.Get("/:sessionId/proxy",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetProxy,
	)

	sessionRoutes.Post("/:sessionId/proxy/test",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.TestProxy,
	)

	// Webhook routes
	sessionRoutes.Get("/:sessionId/webhooks/find",
		r.validationMiddleware.ValidateSessionAccess(),
		r.webhookHandler.FindWebhook,
	)

	sessionRoutes.Post("/:sessionId/webhooks/set",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.WebhookConfigRequest{}),
		r.webhookHandler.SetWebhook,
	)

	sessionRoutes.Get("/:sessionId/webhooks/events",
		r.validationMiddleware.ValidateSessionAccess(),
		r.webhookHandler.GetWebhookEvents,
	)

	sessionRoutes.Post("/:sessionId/messages",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendMessageRequest{}),
		r.messageHandler.SendMessage,
	)

	sessionRoutes.Get("/:sessionId/messages",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidatePaginationParams(),
		r.validationMiddleware.ValidateQuery(&dto.MessageListRequest{}),
		r.messageHandler.GetMessages,
	)

	sessionRoutes.Post("/:sessionId/messages/bulk",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.BulkMessageRequest{}),
		r.messageHandler.SendBulkMessages,
	)

	sessionRoutes.Get("/:sessionId/messages/:messageId/status",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateParams(),
		r.messageHandler.GetMessageStatus,
	)
}

func (r *Router) setupMessageRoutes() {

	sessions := r.app.Group("/sessions/:sessionId")

	// Setup message-specific routes under /messages/
	r.setupMessageSendRoutes(sessions)

	// Setup other categorized routes
	r.setupPresenceRoutes(sessions)
	r.setupContactRoutes(sessions)
	r.setupGroupRoutes(sessions)
	r.setupNewsletterRoutes(sessions)
}

func (r *Router) setupMessageSendRoutes(sessions fiber.Router) {
	messageRoutes := sessions.Group("/messages",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
	)

	// Send message routes
	messageRoutes.Post("/send/text",
		r.validationMiddleware.ValidateJSON(&dto.SendTextRequest{}),
		r.messageHandler.SendText,
	)

	messageRoutes.Post("/send/media",
		r.validationMiddleware.ValidateJSON(&dto.SendMediaRequest{}),
		r.messageHandler.SendMedia,
	)

	messageRoutes.Post("/send/location",
		r.validationMiddleware.ValidateJSON(&dto.SendLocationRequest{}),
		r.messageHandler.SendLocation,
	)

	messageRoutes.Post("/send/contact",
		r.validationMiddleware.ValidateJSON(&dto.SendContactRequest{}),
		r.messageHandler.SendContact,
	)

	messageRoutes.Post("/send/sticker",
		r.validationMiddleware.ValidateJSON(&dto.SendStickerRequest{}),
		r.messageHandler.SendSticker,
	)

	messageRoutes.Post("/send/buttons",
		r.validationMiddleware.ValidateJSON(&dto.SendButtonsRequest{}),
		r.messageHandler.SendButtons,
	)

	messageRoutes.Post("/send/list",
		r.validationMiddleware.ValidateJSON(&dto.SendListRequest{}),
		r.messageHandler.SendList,
	)

	messageRoutes.Post("/send/poll",
		r.validationMiddleware.ValidateJSON(&dto.SendPollRequest{}),
		r.messageHandler.SendPoll,
	)

	messageRoutes.Post("/send/edit",
		r.validationMiddleware.ValidateJSON(&dto.EditMessageRequest{}),
		r.messageHandler.EditMessage,
	)

	// Message interaction routes
	messageRoutes.Post("/react",
		r.validationMiddleware.ValidateJSON(&dto.ReactRequest{}),
		r.messageHandler.ReactToMessage,
	)

	messageRoutes.Post("/delete",
		r.validationMiddleware.ValidateJSON(&dto.DeleteMessageRequest{}),
		r.messageHandler.DeleteMessage,
	)

	// Chat management routes
	messageRoutes.Post("/chat/presence",
		r.validationMiddleware.ValidateJSON(&dto.ChatPresenceRequest{}),
		r.messageHandler.SetChatPresence,
	)

	messageRoutes.Post("/chat/markread",
		r.validationMiddleware.ValidateJSON(&dto.MarkReadRequest{}),
		r.messageHandler.MarkAsRead,
	)

	// Download routes
	messageRoutes.Post("/download/image",
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadImage,
	)

	messageRoutes.Post("/download/video",
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadVideo,
	)

	messageRoutes.Post("/download/audio",
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadAudio,
	)

	messageRoutes.Post("/download/document",
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadDocument,
	)
}

func (r *Router) setupPresenceRoutes(sessions fiber.Router) {
	presenceRoutes := sessions.Group("/presence",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
	)

	presenceRoutes.Post("/set",
		r.validationMiddleware.ValidateJSON(&dto.SessionPresenceRequest{}),
		r.sessionHandler.SetPresence,
	)
}

func (r *Router) setupContactRoutes(sessions fiber.Router) {
	contactRoutes := sessions.Group("/contacts",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
	)

	contactRoutes.Get("/",
		r.sessionHandler.GetContacts,
	)

	contactRoutes.Post("/check",
		r.validationMiddleware.ValidateJSON(&dto.CheckContactRequest{}),
		r.sessionHandler.CheckContacts,
	)

	contactRoutes.Post("/info",
		r.validationMiddleware.ValidateJSON(&dto.ContactInfoRequest{}),
		r.sessionHandler.GetContactInfo,
	)

	contactRoutes.Post("/avatar",
		r.validationMiddleware.ValidateJSON(&dto.ContactAvatarRequest{}),
		r.sessionHandler.GetContactAvatar,
	)
}



func (r *Router) setupNewsletterRoutes(sessions fiber.Router) {
	newsletterRoutes := sessions.Group("/newsletter",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
	)

	newsletterRoutes.Get("/list",
		r.sessionHandler.ListNewsletters,
	)
}

func (r *Router) setupGroupRoutes(sessions fiber.Router) {
	groupRoutes := sessions.Group("/groups",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
	)

	groupRoutes.Post("/create",
		r.validationMiddleware.ValidateJSON(&dto.CreateGroupRequest{}),
		r.groupHandler.CreateGroup,
	)

	groupRoutes.Get("/list",
		r.groupHandler.ListGroups,
	)

	groupRoutes.Post("/info",
		r.validationMiddleware.ValidateJSON(&dto.GroupInfoRequest{}),
		r.groupHandler.GetGroupInfo,
	)

	groupRoutes.Post("/invitelink",
		r.validationMiddleware.ValidateJSON(&dto.GroupInviteLinkRequest{}),
		r.groupHandler.GetInviteLink,
	)

	groupRoutes.Post("/leave",
		r.validationMiddleware.ValidateJSON(&dto.LeaveGroupRequest{}),
		r.groupHandler.LeaveGroup,
	)

	groupRoutes.Post("/photo",
		r.validationMiddleware.ValidateJSON(&dto.SetGroupPhotoRequest{}),
		r.groupHandler.SetGroupPhoto,
	)

	groupRoutes.Post("/photo/remove",
		r.validationMiddleware.ValidateJSON(&dto.GroupPhotoRemoveRequest{}),
		r.groupHandler.RemoveGroupPhoto,
	)

	groupRoutes.Post("/ephemeral",
		r.validationMiddleware.ValidateJSON(&dto.GroupEphemeralRequest{}),
		r.groupHandler.SetGroupEphemeral,
	)

	groupRoutes.Post("/inviteinfo",
		r.validationMiddleware.ValidateJSON(&dto.GroupInviteInfoRequest{}),
		r.groupHandler.GetInviteInfo,
	)

	groupRoutes.Post("/name",
		r.validationMiddleware.ValidateJSON(&dto.SetGroupNameRequest{}),
		r.groupHandler.SetGroupName,
	)

	groupRoutes.Post("/topic",
		r.validationMiddleware.ValidateJSON(&dto.SetGroupTopicRequest{}),
		r.groupHandler.SetGroupTopic,
	)

	groupRoutes.Post("/announce",
		r.validationMiddleware.ValidateJSON(&dto.SetGroupAnnounceRequest{}),
		r.groupHandler.SetGroupAnnounce,
	)

	groupRoutes.Post("/locked",
		r.validationMiddleware.ValidateJSON(&dto.SetGroupLockedRequest{}),
		r.groupHandler.SetGroupLocked,
	)

	groupRoutes.Post("/join",
		r.validationMiddleware.ValidateJSON(&dto.JoinGroupRequest{}),
		r.groupHandler.JoinGroup,
	)
}

func (r *Router) setupWebhookRoutes() {
	// Webhook routes are now under the session scope
	// /sessions/{sessionId}/webhooks/*
}

func (r *Router) setupMediaRoutes() {

	if r.mediaHandler == nil {

		return
	}

	mediaRoutes := r.app.Group("/sessions/:sessionId/media")

	mediaRoutes.Use(
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
	)

	mediaRoutes.Get("/", r.mediaHandler.GetMedia)

	mediaRoutes.Get("/download", r.mediaHandler.DownloadMedia)

	mediaRoutes.Get("/list", r.mediaHandler.ListSessionMedia)

	mediaRoutes.Delete("/", r.mediaHandler.DeleteMedia)
}