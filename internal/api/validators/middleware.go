package validators

import (
	"github.com/gofiber/fiber/v2"
)

// ValidationMiddleware middleware para validação automática
type ValidationMiddleware struct {
	validator *Validator
}

// NewValidationMiddleware cria uma nova instância do middleware de validação
func NewValidationMiddleware() *ValidationMiddleware {
	return &ValidationMiddleware{
		validator: NewValidator(),
	}
}

// ValidateJSON middleware para validar JSON automaticamente
func (vm *ValidationMiddleware) ValidateJSON(structType interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Criar uma nova instância do tipo
		obj := structType
		
		// Validar e fazer bind
		if err := vm.validator.ValidateAndBindJSON(c, obj); err != nil {
			if validationErr, ok := err.(*ValidationErrorResponse); ok {
				return c.Status(validationErr.Status).JSON(validationErr)
			}
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "VALIDATION_ERROR",
					"message": err.Error(),
				})
		}
		
		// Armazenar objeto validado no contexto
		c.Locals("validated_body", obj)
		
		return c.Next()
	}
}

// ValidateQuery middleware para validar query parameters
func (vm *ValidationMiddleware) ValidateQuery(structType interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Criar uma nova instância do tipo
		obj := structType
		
		// Validar query parameters
		if err := vm.validator.ValidateQuery(c, obj); err != nil {
			if validationErr, ok := err.(*ValidationErrorResponse); ok {
				return c.Status(validationErr.Status).JSON(validationErr)
			}
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "VALIDATION_ERROR",
					"message": err.Error(),
				})
		}
		
		// Armazenar objeto validado no contexto
		c.Locals("validated_query", obj)
		
		return c.Next()
	}
}

// ValidateParams middleware para validar parâmetros de rota
func (vm *ValidationMiddleware) ValidateParams() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Validar sessionId se presente
		if sessionID := c.Params("sessionId"); sessionID != "" {
			if err := vm.validator.ValidateSessionID(sessionID); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(&ValidationErrorResponse{
					ErrorCode: "INVALID_PARAM",
					Message:   err.Error(),
					Fields: []ValidationError{{
						Field:   "sessionId",
						Message: err.Error(),
						Value:   sessionID,
					}},
					Code:   "INVALID_SESSION_ID",
					Status: fiber.StatusBadRequest,
				})
			}
		}
		
		// Validar messageId se presente
		if messageID := c.Params("messageId"); messageID != "" {
			if messageID == "" {
				return c.Status(fiber.StatusBadRequest).JSON(&ValidationErrorResponse{
					ErrorCode: "INVALID_PARAM",
					Message:   "Message ID cannot be empty",
					Fields: []ValidationError{{
						Field:   "messageId",
						Message: "Message ID is required",
						Value:   messageID,
					}},
					Code:   "INVALID_MESSAGE_ID",
					Status: fiber.StatusBadRequest,
				})
			}
		}
		
		return c.Next()
	}
}

// GetValidatedBody extrai o objeto validado do contexto
func GetValidatedBody(c *fiber.Ctx) interface{} {
	return c.Locals("validated_body")
}

// GetValidatedQuery extrai os query parameters validados do contexto
func GetValidatedQuery(c *fiber.Ctx) interface{} {
	return c.Locals("validated_query")
}

// ValidateSessionAccess middleware para validar acesso à sessão
func (vm *ValidationMiddleware) ValidateSessionAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Params("sessionId")
		if sessionID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(&ValidationErrorResponse{
				ErrorCode: "MISSING_PARAM",
				Message: "Session ID is required",
				Code:   "MISSING_SESSION_ID",
				Status: fiber.StatusBadRequest,
			})
		}
		
		if err := vm.validator.ValidateSessionID(sessionID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(&ValidationErrorResponse{
				ErrorCode: "INVALID_PARAM",
				Message:   err.Error(),
				Fields: []ValidationError{{
					Field:   "sessionId",
					Message: err.Error(),
					Value:   sessionID,
				}},
				Code:   "INVALID_SESSION_ID",
				Status: fiber.StatusBadRequest,
			})
		}
		
		// Armazenar sessionID validado no contexto
		c.Locals("session_id", sessionID)
		
		return c.Next()
	}
}

// ValidatePaginationParams middleware para validar parâmetros de paginação
func (vm *ValidationMiddleware) ValidatePaginationParams() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)
		
		validLimit, validOffset, err := vm.validator.ValidatePagination(limit, offset)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(&ValidationErrorResponse{
				ErrorCode: "INVALID_PAGINATION",
				Message: err.Error(),
				Code:   "PAGINATION_ERROR",
				Status: fiber.StatusBadRequest,
			})
		}
		
		// Armazenar valores validados no contexto
		c.Locals("pagination_limit", validLimit)
		c.Locals("pagination_offset", validOffset)
		
		return c.Next()
	}
}

// GetValidatedPagination extrai parâmetros de paginação validados
func GetValidatedPagination(c *fiber.Ctx) (int, int) {
	limit := c.Locals("pagination_limit").(int)
	offset := c.Locals("pagination_offset").(int)
	return limit, offset
}

// GetValidatedSessionID extrai sessionID validado do contexto
func GetValidatedSessionID(c *fiber.Ctx) string {
	if sessionID := c.Locals("session_id"); sessionID != nil {
		return sessionID.(string)
	}
	return c.Params("sessionId")
}