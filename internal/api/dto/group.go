package dto

import "time"

// === DTOs PARA OPERAÇÕES DE GRUPO ===

// CreateGroupRequest para criar grupo
type CreateGroupRequest struct {
	Name         string   `json:"name" validate:"required,min=1,max=25"`
	Participants []string `json:"participants" validate:"required,min=1"`
	Description  string   `json:"description,omitempty" validate:"max=512"`
}

// GroupInfoRequest para obter informações do grupo
type GroupInfoRequest struct {
	GroupID string `json:"group_id" validate:"required"`
}

// GroupInviteLinkRequest para obter link de convite
type GroupInviteLinkRequest struct {
	GroupID string `json:"group_id" validate:"required"`
}

// SetGroupPhotoRequest para definir foto do grupo
type SetGroupPhotoRequest struct {
	GroupID string `json:"group_id" validate:"required"`
	Photo   string `json:"photo" validate:"required"` // Base64 ou URL
}

// LeaveGroupRequest para sair do grupo
type LeaveGroupRequest struct {
	GroupID string `json:"group_id" validate:"required"`
}

// SetGroupNameRequest para definir nome do grupo
type SetGroupNameRequest struct {
	GroupID string `json:"group_id" validate:"required"`
	Name    string `json:"name" validate:"required,min=1,max=25"`
}

// SetGroupTopicRequest para definir tópico do grupo
type SetGroupTopicRequest struct {
	GroupID string `json:"group_id" validate:"required"`
	Topic   string `json:"topic" validate:"required,max=512"`
}

// SetGroupAnnounceRequest para configurar anúncios do grupo
type SetGroupAnnounceRequest struct {
	GroupID      string `json:"group_id" validate:"required"`
	AnnounceMode bool   `json:"announce_mode"`
}

// SetGroupLockedRequest para bloquear grupo
type SetGroupLockedRequest struct {
	GroupID string `json:"group_id" validate:"required"`
	Locked  bool   `json:"locked"`
}

// SetGroupEphemeralRequest para configurar timer de desaparecimento
type SetGroupEphemeralRequest struct {
	GroupID   string `json:"group_id" validate:"required"`
	Duration  int64  `json:"duration"` // Em segundos, 0 para desabilitar
}

// JoinGroupRequest para entrar no grupo via link
type JoinGroupRequest struct {
	InviteCode string `json:"invite_code" validate:"required"`
}

// GroupInviteInfoRequest para obter informações do convite
type GroupInviteInfoRequest struct {
	InviteCode string `json:"invite_code" validate:"required"`
}

// UpdateGroupParticipantsRequest para atualizar participantes
type UpdateGroupParticipantsRequest struct {
	GroupID      string   `json:"group_id" validate:"required"`
	Participants []string `json:"participants" validate:"required,min=1"`
	Action       string   `json:"action" validate:"required,oneof=add remove promote demote"`
}

// === ESTRUTURAS DE RESPOSTA ===

// GroupInfo informações do grupo
type GroupInfo struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Topic        string              `json:"topic,omitempty"`
	Owner        string              `json:"owner"`
	Participants []GroupParticipant  `json:"participants"`
	CreatedAt    time.Time           `json:"created_at"`
	AnnounceMode bool                `json:"announce_mode"`
	Locked       bool                `json:"locked"`
	Ephemeral    int64               `json:"ephemeral_duration"`
}

// GroupParticipant participante do grupo
type GroupParticipant struct {
	JID      string `json:"jid"`
	IsAdmin  bool   `json:"is_admin"`
	IsSuperAdmin bool `json:"is_super_admin"`
}

// GroupListResponse lista de grupos
type GroupListResponse struct {
	Groups []GroupInfo `json:"groups"`
	Total  int         `json:"total"`
}

// GroupInviteInfo informações do convite
type GroupInviteInfo struct {
	GroupID     string    `json:"group_id"`
	GroupName   string    `json:"group_name"`
	GroupTopic  string    `json:"group_topic,omitempty"`
	InviteCode  string    `json:"invite_code"`
	Inviter     string    `json:"inviter"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateGroupResponse resposta de criação de grupo
type CreateGroupResponse struct {
	GroupID   string    `json:"group_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Success   bool      `json:"success"`
}

// GroupOperationResponse resposta genérica de operação de grupo
type GroupOperationResponse struct {
	GroupID string `json:"group_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UpdateParticipantsResponse resposta de atualização de participantes
type UpdateParticipantsResponse struct {
	GroupID string                    `json:"group_id"`
	Results []ParticipantUpdateResult `json:"results"`
	Success bool                      `json:"success"`
}

// ParticipantUpdateResult resultado da atualização de um participante
type ParticipantUpdateResult struct {
	JID     string `json:"jid"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
