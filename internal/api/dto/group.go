package dto

import "time"

type CreateGroupRequest struct {
	Name         string   `json:"name" validate:"required,min=1,max=25"`
	Participants []string `json:"participants" validate:"required,min=1"`
	Description  string   `json:"description,omitempty" validate:"max=512"`
}

type GroupInfoRequest struct {
	GroupID string `json:"group_id" validate:"required"`
}

type GroupInviteLinkRequest struct {
	GroupID string `json:"group_id" validate:"required"`
}

type SetGroupPhotoRequest struct {
	GroupID string `json:"group_id" validate:"required"`
	Photo   string `json:"photo" validate:"required"`
}

type LeaveGroupRequest struct {
	GroupID string `json:"group_id" validate:"required"`
}

type SetGroupNameRequest struct {
	GroupID string `json:"group_id" validate:"required"`
	Name    string `json:"name" validate:"required,min=1,max=25"`
}

type SetGroupTopicRequest struct {
	GroupID string `json:"group_id" validate:"required"`
	Topic   string `json:"topic" validate:"required,max=512"`
}

type SetGroupAnnounceRequest struct {
	GroupID      string `json:"group_id" validate:"required"`
	AnnounceMode bool   `json:"announce_mode"`
}

type SetGroupLockedRequest struct {
	GroupID string `json:"group_id" validate:"required"`
	Locked  bool   `json:"locked"`
}

type SetGroupEphemeralRequest struct {
	GroupID  string `json:"group_id" validate:"required"`
	Duration int64  `json:"duration"`
}

type JoinGroupRequest struct {
	InviteCode string `json:"invite_code" validate:"required"`
}

type GroupInviteInfoRequest struct {
	InviteCode string `json:"invite_code" validate:"required"`
}

type UpdateGroupParticipantsRequest struct {
	GroupID      string   `json:"group_id" validate:"required"`
	Participants []string `json:"participants" validate:"required,min=1"`
	Action       string   `json:"action" validate:"required,oneof=add remove promote demote"`
}

type GroupInfo struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Topic        string             `json:"topic,omitempty"`
	Owner        string             `json:"owner"`
	Participants []GroupParticipant `json:"participants"`
	CreatedAt    time.Time          `json:"created_at"`
	AnnounceMode bool               `json:"announce_mode"`
	Locked       bool               `json:"locked"`
	Ephemeral    int64              `json:"ephemeral_duration"`
}

type GroupParticipant struct {
	JID          string `json:"jid"`
	IsAdmin      bool   `json:"is_admin"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

type GroupListResponse struct {
	Groups []GroupInfo `json:"groups"`
	Total  int         `json:"total"`
}

type GroupInviteInfo struct {
	GroupID    string     `json:"group_id"`
	GroupName  string     `json:"group_name"`
	GroupTopic string     `json:"group_topic,omitempty"`
	InviteCode string     `json:"invite_code"`
	Inviter    string     `json:"inviter"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type CreateGroupResponse struct {
	GroupID   string    `json:"group_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Success   bool      `json:"success"`
}

type GroupOperationResponse struct {
	GroupID string `json:"group_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type UpdateParticipantsResponse struct {
	GroupID string                    `json:"group_id"`
	Results []ParticipantUpdateResult `json:"results"`
	Success bool                      `json:"success"`
}

type ParticipantUpdateResult struct {
	JID     string `json:"jid"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
