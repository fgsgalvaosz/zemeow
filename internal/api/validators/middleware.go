package validators

import (
	"github.com/gofiber/fiber/v2"
)

type ValidationMiddleware struct {
	validator *Validator
}

func NewValidationMiddleware() *ValidationMiddleware {
	return &ValidationMiddleware{
		validator: NewValidator(),
	}
}

func (vm *ValidationMiddleware) ValidateJSON(structType interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {

		obj := structType

		if err := vm.validator.ValidateAndBindJSON(c, obj); err != nil {
			if validationErr, ok := err.(*ValidationErrorResponse); ok {
				return c.Status(validationErr.Status).JSON(validationErr)
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "VALIDATION_ERROR",
				"message": err.Error(),
			})
		}

		c.Locals("validated_body", obj)

		return c.Next()
	}
}

func (vm *ValidationMiddleware) ValidateQuery(structType interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {

		obj := structType

		if err := vm.validator.ValidateQuery(c, obj); err != nil {
			if validationErr, ok := err.(*ValidationErrorResponse); ok {
				return c.Status(validationErr.Status).JSON(validationErr)
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "VALIDATION_ERROR",
				"message": err.Error(),
			})
		}

		c.Locals("validated_query", obj)

		return c.Next()
	}
}

func (vm *ValidationMiddleware) ValidateParams() fiber.Handler {
	return func(c *fiber.Ctx) error {

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

func GetValidatedBody(c *fiber.Ctx) interface{} {
	return c.Locals("validated_body")
}

func GetValidatedQuery(c *fiber.Ctx) interface{} {
	return c.Locals("validated_query")
}

func (vm *ValidationMiddleware) ValidateSessionAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Params("sessionId")
		if sessionID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(&ValidationErrorResponse{
				ErrorCode: "MISSING_PARAM",
				Message:   "Session ID is required",
				Code:      "MISSING_SESSION_ID",
				Status:    fiber.StatusBadRequest,
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

		c.Locals("session_id", sessionID)

		return c.Next()
	}
}

func (vm *ValidationMiddleware) ValidatePaginationParams() fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		validLimit, validOffset, err := vm.validator.ValidatePagination(limit, offset)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(&ValidationErrorResponse{
				ErrorCode: "INVALID_PAGINATION",
				Message:   err.Error(),
				Code:      "PAGINATION_ERROR",
				Status:    fiber.StatusBadRequest,
			})
		}

		c.Locals("pagination_limit", validLimit)
		c.Locals("pagination_offset", validOffset)

		return c.Next()
	}
}

func GetValidatedPagination(c *fiber.Ctx) (int, int) {
	limit := c.Locals("pagination_limit").(int)
	offset := c.Locals("pagination_offset").(int)
	return limit, offset
}

func GetValidatedSessionID(c *fiber.Ctx) string {
	if sessionID := c.Locals("session_id"); sessionID != nil {
		return sessionID.(string)
	}
	return c.Params("sessionId")
}
