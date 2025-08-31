package routes

import (
	"github.com/felipe/zemeow/internal/api/dto"
	"github.com/gofiber/fiber/v2"
)


func (r *Router) setupGlobalMiddleware() {
	r.app.Use(r.authMiddleware.CORS())
	r.app.Use(r.authMiddleware.RequestLogger())
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

	messages := r.app.Group("/sessions/:sessionId")


	messageRoutes := messages.Group("/",
		r.authMiddleware.RequireAPIKey(),
		r.validationMiddleware.ValidateParams(),
	)




	messageRoutes.Post("/send/text",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendTextRequest{}),
		r.messageHandler.SendText,
	)


	messageRoutes.Post("/send/media",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendMediaRequest{}),
		r.messageHandler.SendMedia,
	)


	messageRoutes.Post("/send/location",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendLocationRequest{}),
		r.messageHandler.SendLocation,
	)


	messageRoutes.Post("/send/contact",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendContactRequest{}),
		r.messageHandler.SendContact,
	)




	messageRoutes.Post("/send/sticker",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendStickerRequest{}),
		r.messageHandler.SendSticker,
	)




	messageRoutes.Post("/send/buttons",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendButtonsRequest{}),
		r.messageHandler.SendButtons,
	)


	messageRoutes.Post("/send/list",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendListRequest{}),
		r.messageHandler.SendList,
	)


	messageRoutes.Post("/send/poll",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SendPollRequest{}),
		r.messageHandler.SendPoll,
	)


	messageRoutes.Post("/send/edit",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.EditMessageRequest{}),
		r.messageHandler.EditMessage,
	)




	messageRoutes.Post("/react",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ReactRequest{}),
		r.messageHandler.ReactToMessage,
	)


	messageRoutes.Post("/delete",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DeleteMessageRequest{}),
		r.messageHandler.DeleteMessage,
	)




	messageRoutes.Post("/chat/presence",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ChatPresenceRequest{}),
		r.messageHandler.SetChatPresence,
	)


	messageRoutes.Post("/chat/markread",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.MarkReadRequest{}),
		r.messageHandler.MarkAsRead,
	)


	messageRoutes.Post("/download/image",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadImage,
	)


	messageRoutes.Post("/download/video",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadVideo,
	)


	messageRoutes.Post("/download/audio",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadAudio,
	)


	messageRoutes.Post("/download/document",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.DownloadMediaRequest{}),
		r.messageHandler.DownloadDocument,
	)




	messageRoutes.Post("/presence",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SessionPresenceRequest{}),
		r.sessionHandler.SetPresence,
	)


	messageRoutes.Post("/check",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.CheckContactRequest{}),
		r.sessionHandler.CheckContacts,
	)


	messageRoutes.Post("/info",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ContactInfoRequest{}),
		r.sessionHandler.GetContactInfo,
	)


	messageRoutes.Post("/avatar",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ContactAvatarRequest{}),
		r.sessionHandler.GetContactAvatar,
	)


	messageRoutes.Get("/contacts",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetContacts,
	)




	messageRoutes.Post("/proxy",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.ProxyConfigRequest{}),
		r.sessionHandler.ConfigureProxy,
	)


	messageRoutes.Post("/s3/config",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.S3ConfigRequest{}),
		r.sessionHandler.ConfigureS3,
	)


	messageRoutes.Get("/s3/config",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.GetS3Config,
	)


	messageRoutes.Delete("/s3/config",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.DeleteS3Config,
	)


	messageRoutes.Post("/s3/test",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.TestS3Connection,
	)


	messageRoutes.Post("/pairphone",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.PairPhoneRequest{}),
		r.sessionHandler.PairPhone,
	)


	messageRoutes.Get("/newsletter/list",
		r.validationMiddleware.ValidateSessionAccess(),
		r.sessionHandler.ListNewsletters,
	)




	messageRoutes.Post("/group/create",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.CreateGroupRequest{}),
		r.groupHandler.CreateGroup,
	)


	messageRoutes.Get("/group/list",
		r.validationMiddleware.ValidateSessionAccess(),
		r.groupHandler.ListGroups,
	)


	messageRoutes.Post("/group/info",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.GroupInfoRequest{}),
		r.groupHandler.GetGroupInfo,
	)


	messageRoutes.Post("/group/invitelink",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.GroupInviteLinkRequest{}),
		r.groupHandler.GetInviteLink,
	)


	messageRoutes.Post("/group/leave",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.LeaveGroupRequest{}),
		r.groupHandler.LeaveGroup,
	)


	messageRoutes.Post("/group/photo",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SetGroupPhotoRequest{}),
		r.groupHandler.SetGroupPhoto,
	)


	messageRoutes.Post("/group/photo/remove",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.GroupPhotoRemoveRequest{}),
		r.groupHandler.RemoveGroupPhoto,
	)


	messageRoutes.Post("/group/ephemeral",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.GroupEphemeralRequest{}),
		r.groupHandler.SetGroupEphemeral,
	)


	messageRoutes.Post("/group/inviteinfo",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.GroupInviteInfoRequest{}),
		r.groupHandler.GetInviteInfo,
	)




	messageRoutes.Post("/group/name",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SetGroupNameRequest{}),
		r.groupHandler.SetGroupName,
	)


	messageRoutes.Post("/group/topic",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SetGroupTopicRequest{}),
		r.groupHandler.SetGroupTopic,
	)


	messageRoutes.Post("/group/announce",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SetGroupAnnounceRequest{}),
		r.groupHandler.SetGroupAnnounce,
	)


	messageRoutes.Post("/group/locked",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.SetGroupLockedRequest{}),
		r.groupHandler.SetGroupLocked,
	)


	messageRoutes.Post("/group/join",
		r.validationMiddleware.ValidateSessionAccess(),
		r.validationMiddleware.ValidateJSON(&dto.JoinGroupRequest{}),
		r.groupHandler.JoinGroup,
	)
}


func (r *Router) setupWebhookRoutes() {

	webhooks := r.app.Group("/webhooks", r.authMiddleware.RequireGlobalAPIKey())

	// Find webhooks configured for a session
	webhooks.Get("/sessions/:sessionId/find",
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
		r.webhookHandler.FindWebhook,
	)

	// Set/configure webhook for a session
	webhooks.Post("/sessions/:sessionId/set",
		r.validationMiddleware.ValidateParams(),
		r.validationMiddleware.ValidateSessionAccess(),
		r.webhookHandler.SetWebhook,
	)

	// Get list of all available webhook events
	webhooks.Get("/events", r.webhookHandler.GetWebhookEvents)
}
