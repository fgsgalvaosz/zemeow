package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/api/middleware"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/media"
)

type MediaHandler struct {
	mediaService *media.MediaService
	messageRepo  repositories.MessageRepository
	logger       logger.Logger
}

func NewMediaHandler(mediaService *media.MediaService, messageRepo repositories.MessageRepository) *MediaHandler {
	return &MediaHandler{
		mediaService: mediaService,
		messageRepo:  messageRepo,
		logger:       logger.GetWithSession("media_handler"),
	}
}

// @Summary Obter mídia
// @Description Obtém uma mídia armazenada no MinIO pelo path
// @Tags media
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param path query string true "Path da mídia no MinIO"
// @Success 200 {object} map[string]interface{} "Informações da mídia"
// @Failure 400 {object} map[string]interface{} "Parâmetros inválidos"
// @Failure 404 {object} map[string]interface{} "Mídia não encontrada"
// @Router /sessions/{sessionId}/media [get]
func (h *MediaHandler) GetMedia(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	path := c.Query("path")
	if path == "" {
		return h.sendError(c, "Media path is required", "MISSING_PATH", fiber.StatusBadRequest)
	}

	if !h.validateSessionPath(sessionID, path) {
		return h.sendError(c, "Access denied to this media", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	mediaInfo, err := h.mediaService.GetMedia(context.Background(), path)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("path", path).Msg("Failed to get media")
		return h.sendError(c, "Failed to get media", "MEDIA_NOT_FOUND", fiber.StatusNotFound)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"media":   mediaInfo,
	})
}

// @Summary Baixar mídia
// @Description Baixa uma mídia armazenada no MinIO
// @Tags media
// @Accept json
// @Produce application/octet-stream
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param path query string true "Path da mídia no MinIO"
// @Success 200 {file} file "Arquivo da mídia"
// @Failure 400 {object} map[string]interface{} "Parâmetros inválidos"
// @Failure 404 {object} map[string]interface{} "Mídia não encontrada"
// @Router /sessions/{sessionId}/media/download [get]
func (h *MediaHandler) DownloadMedia(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	path := c.Query("path")
	if path == "" {
		return h.sendError(c, "Media path is required", "MISSING_PATH", fiber.StatusBadRequest)
	}

	if !h.validateSessionPath(sessionID, path) {
		return h.sendError(c, "Access denied to this media", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	mediaInfo, err := h.mediaService.GetMedia(context.Background(), path)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("path", path).Msg("Failed to get media info")
		return h.sendError(c, "Failed to get media", "MEDIA_NOT_FOUND", fiber.StatusNotFound)
	}

	data, err := h.mediaService.DownloadMedia(context.Background(), path)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("path", path).Msg("Failed to download media")
		return h.sendError(c, "Failed to download media", "DOWNLOAD_FAILED", fiber.StatusInternalServerError)
	}

	c.Set("Content-Type", mediaInfo.ContentType)
	c.Set("Content-Length", strconv.FormatInt(mediaInfo.Size, 10))
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", mediaInfo.FileName))

	return c.Send(data)
}

// @Summary Listar mídias da sessão
// @Description Lista todas as mídias de uma sessão com paginação
// @Tags media
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param page query int false "Página (padrão: 1)"
// @Param limit query int false "Limite por página (padrão: 50, máximo: 100)"
// @Param type query string false "Filtrar por tipo de mídia (image, video, audio, document)"
// @Param direction query string false "Filtrar por direção (incoming, outgoing)"
// @Success 200 {object} map[string]interface{} "Lista de mídias"
// @Failure 400 {object} map[string]interface{} "Parâmetros inválidos"
// @Router /sessions/{sessionId}/media/list [get]
func (h *MediaHandler) ListSessionMedia(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}

	mediaType := c.Query("type")
	direction := c.Query("direction")

	messages, total, err := h.messageRepo.GetSessionMediaMessages(sessionID, page, limit, mediaType, direction)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get session media")
		return h.sendError(c, "Failed to get media list", "QUERY_FAILED", fiber.StatusInternalServerError)
	}

	totalPages := (total + limit - 1) / limit

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"messages": messages,
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
			},
			"filters": fiber.Map{
				"type":      mediaType,
				"direction": direction,
			},
		},
	})
}

// @Summary Deletar mídia
// @Description Remove uma mídia do MinIO e atualiza a mensagem
// @Tags media
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param sessionId path string true "ID da sessão"
// @Param path query string true "Path da mídia no MinIO"
// @Success 200 {object} map[string]interface{} "Mídia removida com sucesso"
// @Failure 400 {object} map[string]interface{} "Parâmetros inválidos"
// @Failure 404 {object} map[string]interface{} "Mídia não encontrada"
// @Router /sessions/{sessionId}/media [delete]
func (h *MediaHandler) DeleteMedia(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return h.sendError(c, "Session ID is required", "MISSING_SESSION_ID", fiber.StatusBadRequest)
	}

	path := c.Query("path")
	if path == "" {
		return h.sendError(c, "Media path is required", "MISSING_PATH", fiber.StatusBadRequest)
	}

	if !h.validateSessionPath(sessionID, path) {
		return h.sendError(c, "Access denied to this media", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	auth := middleware.GetAuthContext(c)
	if auth == nil || (!auth.IsGlobalKey && auth.SessionID != sessionID) {
		return h.sendError(c, "Access denied", "ACCESS_DENIED", fiber.StatusForbidden)
	}

	err := h.mediaService.DeleteMedia(context.Background(), path)
	if err != nil {
		h.logger.Error().Err(err).Str("session_id", sessionID).Str("path", path).Msg("Failed to delete media")
		return h.sendError(c, "Failed to delete media", "DELETE_FAILED", fiber.StatusInternalServerError)
	}

	err = h.messageRepo.ClearMinIOReferences(path)
	if err != nil {
		h.logger.Warn().Err(err).Str("path", path).Msg("Failed to clear MinIO references in database")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Media deleted successfully",
	})
}

func (h *MediaHandler) validateSessionPath(sessionID, path string) bool {
	expectedPrefix := fmt.Sprintf("sessions/%s/", sessionID)
	return len(path) > len(expectedPrefix) && path[:len(expectedPrefix)] == expectedPrefix
}

func (h *MediaHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	return c.Status(status).JSON(fiber.Map{
		"success":   false,
		"error":     code,
		"message":   message,
		"timestamp": time.Now().Unix(),
	})
}
