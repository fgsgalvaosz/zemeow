package utils

import (
	"fmt"
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow/types"
)

// PhoneValidator fornece utilitários para validação de números de telefone
type PhoneValidator struct{}

// NewPhoneValidator cria uma nova instância do validador
func NewPhoneValidator() *PhoneValidator {
	return &PhoneValidator{}
}

// CleanPhone limpa um número de telefone removendo caracteres especiais
func (pv *PhoneValidator) CleanPhone(phone string) string {
	// Remover caracteres especiais comuns
	cleaned := strings.ReplaceAll(phone, "+", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	
	return cleaned
}

// IsValidPhone verifica se um número de telefone é válido
func (pv *PhoneValidator) IsValidPhone(phone string) bool {
	cleaned := pv.CleanPhone(phone)
	
	// Verificar se contém apenas dígitos
	if matched, _ := regexp.MatchString(`^\d+$`, cleaned); !matched {
		return false
	}
	
	// Verificar comprimento (mínimo 10, máximo 15 dígitos)
	if len(cleaned) < 10 || len(cleaned) > 15 {
		return false
	}
	
	return true
}

// ParseToJID converte um número de telefone para JID do WhatsApp
func (pv *PhoneValidator) ParseToJID(phone string) (types.JID, error) {
	cleaned := pv.CleanPhone(phone)
	
	if !pv.IsValidPhone(phone) {
		return types.EmptyJID, &PhoneValidationError{
			Phone:   phone,
			Message: "Invalid phone number format",
		}
	}
	
	return types.ParseJID(cleaned + "@s.whatsapp.net")
}

// ParseGroupJID converte um ID de grupo para JID do WhatsApp
func (pv *PhoneValidator) ParseGroupJID(groupID string) (types.JID, error) {
	// Se já contém @g.us, usar diretamente
	if strings.Contains(groupID, "@g.us") {
		return types.ParseJID(groupID)
	}
	
	// Se não contém, assumir que é apenas o ID e adicionar @g.us
	return types.ParseJID(groupID + "@g.us")
}

// ValidatePhoneList valida uma lista de números de telefone
func (pv *PhoneValidator) ValidatePhoneList(phones []string) (valid []string, invalid []string) {
	for _, phone := range phones {
		if pv.IsValidPhone(phone) {
			valid = append(valid, phone)
		} else {
			invalid = append(invalid, phone)
		}
	}
	return valid, invalid
}

// ConvertPhonestoJIDs converte uma lista de telefones para JIDs
func (pv *PhoneValidator) ConvertPhonestoJIDs(phones []string) ([]types.JID, []string, error) {
	var jids []types.JID
	var invalidPhones []string
	
	for _, phone := range phones {
		jid, err := pv.ParseToJID(phone)
		if err != nil {
			invalidPhones = append(invalidPhones, phone)
			continue
		}
		jids = append(jids, jid)
	}
	
	return jids, invalidPhones, nil
}

// FormatPhoneForDisplay formata um número de telefone para exibição
func (pv *PhoneValidator) FormatPhoneForDisplay(phone string) string {
	cleaned := pv.CleanPhone(phone)
	
	// Se tem código do país brasileiro (55)
	if strings.HasPrefix(cleaned, "55") && len(cleaned) >= 13 {
		// Formato: +55 (11) 99999-9999
		return fmt.Sprintf("+55 (%s) %s-%s", 
			cleaned[2:4], 
			cleaned[4:9], 
			cleaned[9:])
	}
	
	// Formato genérico com + no início
	if len(cleaned) > 10 {
		return "+" + cleaned
	}
	
	return cleaned
}

// GetCountryCode extrai o código do país de um número de telefone
func (pv *PhoneValidator) GetCountryCode(phone string) string {
	cleaned := pv.CleanPhone(phone)
	
	// Códigos de país comuns (simplificado)
	countryCodes := map[string]string{
		"55": "BR", // Brasil
		"1":  "US", // Estados Unidos/Canadá
		"44": "GB", // Reino Unido
		"49": "DE", // Alemanha
		"33": "FR", // França
		"39": "IT", // Itália
		"34": "ES", // Espanha
		"351": "PT", // Portugal
	}
	
	// Verificar códigos de 3 dígitos primeiro
	if len(cleaned) >= 3 {
		if country, exists := countryCodes[cleaned[:3]]; exists {
			return country
		}
	}
	
	// Verificar códigos de 2 dígitos
	if len(cleaned) >= 2 {
		if country, exists := countryCodes[cleaned[:2]]; exists {
			return country
		}
	}
	
	// Verificar códigos de 1 dígito
	if len(cleaned) >= 1 {
		if country, exists := countryCodes[cleaned[:1]]; exists {
			return country
		}
	}
	
	return "UNKNOWN"
}

// PhoneValidationError erro customizado para validação de telefone
type PhoneValidationError struct {
	Phone   string
	Message string
}

func (e *PhoneValidationError) Error() string {
	return fmt.Sprintf("phone validation error for '%s': %s", e.Phone, e.Message)
}

// PhoneInfo informações detalhadas sobre um número de telefone
type PhoneInfo struct {
	Original    string `json:"original"`
	Cleaned     string `json:"cleaned"`
	Formatted   string `json:"formatted"`
	CountryCode string `json:"country_code"`
	IsValid     bool   `json:"is_valid"`
	JID         string `json:"jid,omitempty"`
}

// GetPhoneInfo retorna informações detalhadas sobre um número de telefone
func (pv *PhoneValidator) GetPhoneInfo(phone string) *PhoneInfo {
	info := &PhoneInfo{
		Original:    phone,
		Cleaned:     pv.CleanPhone(phone),
		Formatted:   pv.FormatPhoneForDisplay(phone),
		CountryCode: pv.GetCountryCode(phone),
		IsValid:     pv.IsValidPhone(phone),
	}
	
	if info.IsValid {
		if jid, err := pv.ParseToJID(phone); err == nil {
			info.JID = jid.String()
		}
	}
	
	return info
}

// Instância global do validador
var DefaultPhoneValidator = NewPhoneValidator()

// Funções de conveniência
func CleanPhone(phone string) string {
	return DefaultPhoneValidator.CleanPhone(phone)
}

func IsValidPhone(phone string) bool {
	return DefaultPhoneValidator.IsValidPhone(phone)
}

func ParseToJID(phone string) (types.JID, error) {
	return DefaultPhoneValidator.ParseToJID(phone)
}

func ParseGroupJID(groupID string) (types.JID, error) {
	return DefaultPhoneValidator.ParseGroupJID(groupID)
}
