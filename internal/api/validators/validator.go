package validators

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Validator struct {
	validate *validator.Validate
}

type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

type ValidationErrorResponse struct {
	ErrorCode string            `json:"error"`
	Message   string            `json:"message"`
	Fields    []ValidationError `json:"fields"`
	Code      string            `json:"code"`
	Status    int               `json:"status"`
}

func (v *ValidationErrorResponse) Error() string {
	return v.Message
}

func NewValidator() *Validator {
	v := validator.New()

	registerCustomValidations(v)

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	return &Validator{validate: v}
}

func registerCustomValidations(v *validator.Validate) {

	v.RegisterValidation("e164", validateE164)

	v.RegisterValidation("latitude", validateLatitude)

	v.RegisterValidation("longitude", validateLongitude)

	v.RegisterValidation("session_id", validateSessionID)

	v.RegisterValidation("api_key", validateAPIKey)
}

func validateE164(fl validator.FieldLevel) bool {
	phone := fl.Field().String()

	regex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	return regex.MatchString(phone)
}

func validateLatitude(fl validator.FieldLevel) bool {
	lat := fl.Field().Float()
	return lat >= -90 && lat <= 90
}

func validateLongitude(fl validator.FieldLevel) bool {
	lng := fl.Field().Float()
	return lng >= -180 && lng <= 180
}

func validateSessionID(fl validator.FieldLevel) bool {
	sessionID := fl.Field().String()

	regex := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)
	return regex.MatchString(sessionID)
}

func validateAPIKey(fl validator.FieldLevel) bool {
	apiKey := fl.Field().String()

	return len(apiKey) >= 32
}

func (v *Validator) ValidateStruct(s interface{}) error {
	return v.validate.Struct(s)
}

func (v *Validator) ValidateAndBindJSON(c *fiber.Ctx, obj interface{}) error {

	if err := c.BodyParser(obj); err != nil {
		return &ValidationErrorResponse{
			ErrorCode: "INVALID_JSON",
			Message:   "Invalid JSON format",
			Code:      "PARSE_ERROR",
			Status:    fiber.StatusBadRequest,
		}
	}

	if err := v.ValidateStruct(obj); err != nil {
		return v.formatValidationError(err)
	}

	return nil
}

func (v *Validator) ValidateQuery(c *fiber.Ctx, obj interface{}) error {

	if err := c.QueryParser(obj); err != nil {
		return &ValidationErrorResponse{
			ErrorCode: "INVALID_QUERY",
			Message:   "Invalid query parameters",
			Code:      "QUERY_PARSE_ERROR",
			Status:    fiber.StatusBadRequest,
		}
	}

	if err := v.ValidateStruct(obj); err != nil {
		return v.formatValidationError(err)
	}

	return nil
}

func (v *Validator) formatValidationError(err error) *ValidationErrorResponse {
	var validationErrors []ValidationError

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, err := range validationErrs {
			validationErrors = append(validationErrors, ValidationError{
				Field:   err.Field(),
				Tag:     err.Tag(),
				Message: getValidationMessage(err),
				Value:   fmt.Sprintf("%v", err.Value()),
			})
		}
	}

	return &ValidationErrorResponse{
		ErrorCode: "VALIDATION_ERROR",
		Message:   "Request validation failed",
		Fields:    validationErrors,
		Code:      "VALIDATION_FAILED",
		Status:    fiber.StatusBadRequest,
	}
}

func getValidationMessage(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()
	param := err.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("Field '%s' is required", field)
	case "email":
		return fmt.Sprintf("Field '%s' must be a valid email address", field)
	case "url":
		return fmt.Sprintf("Field '%s' must be a valid URL", field)
	case "min":
		return fmt.Sprintf("Field '%s' must be at least %s characters long", field, param)
	case "max":
		return fmt.Sprintf("Field '%s' must be at most %s characters long", field, param)
	case "len":
		return fmt.Sprintf("Field '%s' must be exactly %s characters long", field, param)
	case "alphanum":
		return fmt.Sprintf("Field '%s' must contain only alphanumeric characters", field)
	case "numeric":
		return fmt.Sprintf("Field '%s' must be numeric", field)
	case "e164":
		return fmt.Sprintf("Field '%s' must be a valid phone number in E.164 format", field)
	case "latitude":
		return fmt.Sprintf("Field '%s' must be a valid latitude (-90 to 90)", field)
	case "longitude":
		return fmt.Sprintf("Field '%s' must be a valid longitude (-180 to 180)", field)
	case "session_id":
		return fmt.Sprintf("Field '%s' must be a valid session ID (3-50 alphanumeric characters)", field)
	case "api_key":
		return fmt.Sprintf("Field '%s' must be a valid API key (minimum 32 characters)", field)
	case "oneof":
		return fmt.Sprintf("Field '%s' must be one of: %s", field, param)
	case "required_if":
		return fmt.Sprintf("Field '%s' is required when %s", field, param)
	case "datetime":
		return fmt.Sprintf("Field '%s' must be a valid date in format %s", field, param)
	default:
		return fmt.Sprintf("Field '%s' is invalid", field)
	}
}

func (v *Validator) ValidateSessionID(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}

	regex := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)
	if !regex.MatchString(sessionID) {
		return fmt.Errorf("session ID must be 3-50 alphanumeric characters, underscores, or hyphens")
	}

	return nil
}

func (v *Validator) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	if len(apiKey) < 32 {
		return fmt.Errorf("API key must be at least 32 characters long")
	}

	return nil
}

func (v *Validator) ValidatePhoneNumber(phone string) error {
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}

	regex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !regex.MatchString(phone) {
		return fmt.Errorf("phone number must be in E.164 format (e.g., +5511999999999)")
	}

	return nil
}

func (v *Validator) ValidatePagination(limit, offset int) (int, int, error) {

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	if limit > 100 {
		limit = 100
	}

	return limit, offset, nil
}

func (v *Validator) ValidateDateRange(dateFrom, dateTo string) (time.Time, time.Time, error) {
	var from, to time.Time
	var err error

	layout := "2006-01-02"

	if dateFrom != "" {
		from, err = time.Parse(layout, dateFrom)
		if err != nil {
			return from, to, fmt.Errorf("invalid date_from format, use YYYY-MM-DD")
		}
	}

	if dateTo != "" {
		to, err = time.Parse(layout, dateTo)
		if err != nil {
			return from, to, fmt.Errorf("invalid date_to format, use YYYY-MM-DD")
		}
	}

	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return from, to, fmt.Errorf("date_from cannot be after date_to")
	}

	return from, to, nil
}
