package utils

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// ErrorResponse representa uma resposta de erro padronizada
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Error     ErrorDetails           `json:"error"`
	Timestamp int64                  `json:"timestamp"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

// ErrorDetails contém os detalhes do erro
type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SuccessResponse representa uma resposta de sucesso padronizada
type SuccessResponse struct {
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

// SendError envia uma resposta de erro padronizada
func SendError(c *fiber.Ctx, message, code string, status int) error {
	response := ErrorResponse{
		Success: false,
		Error: ErrorDetails{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now().Unix(),
	}

	return c.Status(status).JSON(response)
}

// SendErrorWithMeta envia uma resposta de erro com metadados adicionais
func SendErrorWithMeta(c *fiber.Ctx, message, code string, status int, meta map[string]interface{}) error {
	response := ErrorResponse{
		Success: false,
		Error: ErrorDetails{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now().Unix(),
		Meta:      meta,
	}

	return c.Status(status).JSON(response)
}

// SendSuccess envia uma resposta de sucesso padronizada
func SendSuccess(c *fiber.Ctx, data interface{}, message string) error {
	response := SuccessResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	return c.JSON(response)
}

// SendSuccessWithMeta envia uma resposta de sucesso com metadados adicionais
func SendSuccessWithMeta(c *fiber.Ctx, data interface{}, message string, meta map[string]interface{}) error {
	response := SuccessResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().Unix(),
		Meta:      meta,
	}

	return c.JSON(response)
}

// Códigos de erro comuns
const (
	ErrCodeValidation      = "VALIDATION_ERROR"
	ErrCodeAuthentication  = "AUTHENTICATION_ERROR"
	ErrCodeAuthorization   = "AUTHORIZATION_ERROR"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeInternalError   = "INTERNAL_ERROR"
	ErrCodeBadRequest      = "BAD_REQUEST"
	ErrCodeSessionNotReady = "SESSION_NOT_READY"
	ErrCodeAccessDenied    = "ACCESS_DENIED"
	ErrCodeInvalidJSON     = "INVALID_JSON"
	ErrCodeSendFailed      = "SEND_FAILED"
)

// Funções de conveniência para erros comuns
func SendValidationError(c *fiber.Ctx, message string) error {
	return SendError(c, message, ErrCodeValidation, fiber.StatusBadRequest)
}

func SendAuthenticationError(c *fiber.Ctx, message string) error {
	return SendError(c, message, ErrCodeAuthentication, fiber.StatusUnauthorized)
}

func SendAuthorizationError(c *fiber.Ctx, message string) error {
	return SendError(c, message, ErrCodeAuthorization, fiber.StatusForbidden)
}

func SendNotFoundError(c *fiber.Ctx, message string) error {
	return SendError(c, message, ErrCodeNotFound, fiber.StatusNotFound)
}

func SendInternalError(c *fiber.Ctx, message string) error {
	return SendError(c, message, ErrCodeInternalError, fiber.StatusInternalServerError)
}

func SendAccessDeniedError(c *fiber.Ctx) error {
	return SendError(c, "Access denied", ErrCodeAccessDenied, fiber.StatusForbidden)
}

func SendInvalidJSONError(c *fiber.Ctx) error {
	return SendError(c, "Invalid request body", ErrCodeInvalidJSON, fiber.StatusBadRequest)
}
