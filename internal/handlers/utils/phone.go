package utils

import (
	"fmt"
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow/types"
)

type PhoneValidator struct{}

func NewPhoneValidator() *PhoneValidator {
	return &PhoneValidator{}
}

func (pv *PhoneValidator) CleanPhone(phone string) string {

	cleaned := strings.ReplaceAll(phone, "+", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")

	return cleaned
}

func (pv *PhoneValidator) IsValidPhone(phone string) bool {
	cleaned := pv.CleanPhone(phone)

	if matched, _ := regexp.MatchString(`^\d+$`, cleaned); !matched {
		return false
	}

	if len(cleaned) < 10 || len(cleaned) > 15 {
		return false
	}

	return true
}

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

func (pv *PhoneValidator) ParseGroupJID(groupID string) (types.JID, error) {

	if strings.Contains(groupID, "@g.us") {
		return types.ParseJID(groupID)
	}

	return types.ParseJID(groupID + "@g.us")
}

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

func (pv *PhoneValidator) FormatPhoneForDisplay(phone string) string {
	cleaned := pv.CleanPhone(phone)

	if strings.HasPrefix(cleaned, "55") && len(cleaned) >= 13 {

		return fmt.Sprintf("+55 (%s) %s-%s",
			cleaned[2:4],
			cleaned[4:9],
			cleaned[9:])
	}

	if len(cleaned) > 10 {
		return "+" + cleaned
	}

	return cleaned
}

func (pv *PhoneValidator) GetCountryCode(phone string) string {
	cleaned := pv.CleanPhone(phone)

	countryCodes := map[string]string{
		"55":  "BR",
		"1":   "US",
		"44":  "GB",
		"49":  "DE",
		"33":  "FR",
		"39":  "IT",
		"34":  "ES",
		"351": "PT",
	}

	if len(cleaned) >= 3 {
		if country, exists := countryCodes[cleaned[:3]]; exists {
			return country
		}
	}

	if len(cleaned) >= 2 {
		if country, exists := countryCodes[cleaned[:2]]; exists {
			return country
		}
	}

	if len(cleaned) >= 1 {
		if country, exists := countryCodes[cleaned[:1]]; exists {
			return country
		}
	}

	return "UNKNOWN"
}

type PhoneValidationError struct {
	Phone   string
	Message string
}

func (e *PhoneValidationError) Error() string {
	return fmt.Sprintf("phone validation error for '%s': %s", e.Phone, e.Message)
}

type PhoneInfo struct {
	Original    string `json:"original"`
	Cleaned     string `json:"cleaned"`
	Formatted   string `json:"formatted"`
	CountryCode string `json:"country_code"`
	IsValid     bool   `json:"is_valid"`
	JID         string `json:"jid,omitempty"`
}

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

var DefaultPhoneValidator = NewPhoneValidator()

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
