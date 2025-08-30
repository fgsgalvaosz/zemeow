package dto




type ProxyConfigRequest struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type,omitempty" validate:"omitempty,oneof=http https socks5"`
	Host     string `json:"host,omitempty" validate:"omitempty,hostname_rfc1123"`
	Port     int    `json:"port,omitempty" validate:"omitempty,min=1,max=65535"`
	Username string `json:"username,omitempty" validate:"omitempty,max=255"`
	Password string `json:"password,omitempty" validate:"omitempty,max=255"`
}


type S3ConfigRequest struct {
	Enabled         bool   `json:"enabled"`
	Bucket          string `json:"bucket,omitempty" validate:"omitempty,min=3,max=63"`
	Region          string `json:"region,omitempty" validate:"omitempty,min=2,max=50"`
	AccessKeyID     string `json:"access_key_id,omitempty" validate:"omitempty,min=16,max=128"`
	SecretAccessKey string `json:"secret_access_key,omitempty" validate:"omitempty,min=16,max=128"`
	Endpoint        string `json:"endpoint,omitempty" validate:"omitempty,url"`
	ForcePathStyle  bool   `json:"force_path_style,omitempty"`
}


type PairPhoneRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"`
}




type ProxyConfigResponse struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type,omitempty"`
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	Status  string `json:"status"` // connected, disconnected, error
}


type S3ConfigResponse struct {
	Enabled    bool   `json:"enabled"`
	Bucket     string `json:"bucket,omitempty"`
	Region     string `json:"region,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	Status     string `json:"status"` // connected, disconnected, error
	LastTested string `json:"last_tested,omitempty"`
}


type S3TestResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Latency     int64  `json:"latency_ms,omitempty"`
	TestedAt    string `json:"tested_at"`
	BucketInfo  *S3BucketInfo `json:"bucket_info,omitempty"`
}


type S3BucketInfo struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	CreationDate string `json:"creation_date,omitempty"`
	Permissions  string `json:"permissions"` // read, write, read-write, none
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
	Success     bool   `json:"success"`
	GroupID     string `json:"group_id,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
	GroupTopic  string `json:"group_topic,omitempty"`
	InviteCode  string `json:"invite_code"`
	Inviter     string `json:"inviter,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	Participants int   `json:"participants,omitempty"`
	IsExpired   bool   `json:"is_expired"`
	CanJoin     bool   `json:"can_join"`
}


type GroupEphemeralRequest struct {
	GroupID  string `json:"group_id" validate:"required"`
	Duration int64  `json:"duration" validate:"min=0,max=31536000"` // 0 a 1 ano em segundos
}


type GroupPhotoRemoveRequest struct {
	GroupID string `json:"group_id" validate:"required"`
}
