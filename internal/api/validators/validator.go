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

// Validator estrutura principal do validador
type Validator struct {
	validate *validator.Validate
}

// ValidationError estrutura de erro de validação
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationErrorResponse resposta de erro de validação
type ValidationErrorResponse struct {
	ErrorCode string            `json:"error"`
	Message   string            `json:"message"`
	Fields    []ValidationError `json:"fields"`
	Code      string            `json:"code"`
	Status    int               `json:"status"`
}

// Error implementa a interface error
func (v *ValidationErrorResponse) Error() string {
	return v.Message
}

// NewValidator cria uma nova instância do validador
func NewValidator() *Validator {
	v := validator.New()
	
	// Registrar validações customizadas
	registerCustomValidations(v)
	
	// Configurar nomes de campo usando JSON tags
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

// registerCustomValidations registra validações customizadas
func registerCustomValidations(v *validator.Validate) {
	// Validação para número de telefone E.164
	v.RegisterValidation("e164", validateE164)
	
	// Validação para latitude
	v.RegisterValidation("latitude", validateLatitude)
	
	// Validação para longitude
	v.RegisterValidation("longitude", validateLongitude)
	
	// Validação para SessionID
	v.RegisterValidation("session_id", validateSessionID)
	
	// Validação para API Key
	v.RegisterValidation("api_key", validateAPIKey)
}

// validateE164 valida formato de telefone E.164
func validateE164(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	// Regex básico para E.164: +{1-3 dígitos de país}{até 14 dígitos}
	regex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	return regex.MatchString(phone)
}

// validateLatitude valida latitude (-90 a 90)
func validateLatitude(fl validator.FieldLevel) bool {
	lat := fl.Field().Float()
	return lat >= -90 && lat <= 90
}

// validateLongitude valida longitude (-180 a 180)
func validateLongitude(fl validator.FieldLevel) bool {
	lng := fl.Field().Float()
	return lng >= -180 && lng <= 180
}

// validateSessionID valida formato de SessionID
func validateSessionID(fl validator.FieldLevel) bool {
	sessionID := fl.Field().String()
	// SessionID deve ser alfanumérico com 3-50 caracteres
	regex := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)
	return regex.MatchString(sessionID)
}

// validateAPIKey valida formato de API Key
func validateAPIKey(fl validator.FieldLevel) bool {
	apiKey := fl.Field().String()
	// API Key deve ter pelo menos 32 caracteres
	return len(apiKey) >= 32
}

// ValidateStruct valida uma estrutura
func (v *Validator) ValidateStruct(s interface{}) error {
	return v.validate.Struct(s)
}

// ValidateAndBindJSON valida e faz bind do JSON da requisição
func (v *Validator) ValidateAndBindJSON(c *fiber.Ctx, obj interface{}) error {
	// Fazer parse do JSON
	if err := c.BodyParser(obj); err != nil {
		return &ValidationErrorResponse{
			ErrorCode: "INVALID_JSON",
			Message:   "Invalid JSON format",
			Code:      "PARSE_ERROR",
			Status:    fiber.StatusBadRequest,
		}
	}

	// Validar estrutura
	if err := v.ValidateStruct(obj); err != nil {
		return v.formatValidationError(err)
	}

	return nil
}

// ValidateQuery valida parâmetros de query
func (v *Validator) ValidateQuery(c *fiber.Ctx, obj interface{}) error {
	// Fazer parse dos query parameters
	if err := c.QueryParser(obj); err != nil {
		return &ValidationErrorResponse{
			ErrorCode: "INVALID_QUERY",
			Message:   "Invalid query parameters",
			Code:      "QUERY_PARSE_ERROR",
			Status:    fiber.StatusBadRequest,
		}
	}

	// Validar estrutura
	if err := v.ValidateStruct(obj); err != nil {
		return v.formatValidationError(err)
	}

	return nil
}

// formatValidationError formata erros de validação
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

// getValidationMessage retorna mensagem de erro personalizada
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

// ValidateSessionID valida um Session ID
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

// ValidateAPIKey valida uma API Key
func (v *Validator) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	
	if len(apiKey) < 32 {
		return fmt.Errorf("API key must be at least 32 characters long")
	}
	
	return nil
}

// ValidatePhoneNumber valida um número de telefone
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

// ValidatePagination valida parâmetros de paginação
func (v *Validator) ValidatePagination(limit, offset int) (int, int, error) {
	// Valores padrão
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	
	// Limites máximos
	if limit > 100 {
		limit = 100
	}
	
	return limit, offset, nil
}

// ValidateDateRange valida um intervalo de datas
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