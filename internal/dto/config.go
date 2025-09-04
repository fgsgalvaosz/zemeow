package dto

type ProxyConfigRequest struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type,omitempty" validate:"omitempty,oneof=http https socks5"`
	Host     string `json:"host,omitempty" validate:"omitempty,hostname_rfc1123"`
	Port     int    `json:"port,omitempty" validate:"omitempty,min=1,max=65535"`
	Username string `json:"username,omitempty" validate:"omitempty,max=255"`
	Password string `json:"password,omitempty" validate:"omitempty,max=255"`
}



type PairPhoneRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"`
}

type ProxyConfigResponse struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type,omitempty"`
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	Status  string `json:"status"`
}



type PairPhoneResponse struct {
	Success   bool   `json:"success"`
	Phone     string `json:"phone"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type NewsletterInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Invite      string `json:"invite,omitempty"`
	Handle      string `json:"handle,omitempty"`
	Picture     string `json:"picture,omitempty"`
	Preview     string `json:"preview,omitempty"`
	Verified    bool   `json:"verified"`
	Subscribers int64  `json:"subscribers,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type NewsletterListResponse struct {
	Newsletters []NewsletterInfo `json:"newsletters"`
	Total       int              `json:"total"`
	Timestamp   int64            `json:"timestamp"`
}

type GroupInviteInfoResponse struct {
	Success      bool   `json:"success"`
	GroupID      string `json:"group_id,omitempty"`
	GroupName    string `json:"group_name,omitempty"`
	GroupTopic   string `json:"group_topic,omitempty"`
	InviteCode   string `json:"invite_code"`
	Inviter      string `json:"inviter,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	Participants int    `json:"participants,omitempty"`
	IsExpired    bool   `json:"is_expired"`
	CanJoin      bool   `json:"can_join"`
}

type GroupEphemeralRequest struct {
	GroupID  string `json:"group_id" validate:"required"`
	Duration int64  `json:"duration" validate:"min=0,max=31536000"`
}

type GroupPhotoRemoveRequest struct {
	GroupID string `json:"group_id" validate:"required"`
}
