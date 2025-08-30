package dto

import "time"

// BaseResponse estrutura base para respostas da API
type BaseResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Code      string      `json:"code,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// ErrorResponse estrutura para respostas de erro
type ErrorResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Code      string `json:"code,omitempty"`
	Status    int    `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// PaginationResponse estrutura para respostas paginadas
type PaginationResponse struct {
	BaseResponse
	Pagination PaginationInfo `json:"pagination"`
}

// PaginationInfo informações de paginação
type PaginationInfo struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	Total     int `json:"total"`
	TotalPage int `json:"total_pages"`
	HasNext   bool `json:"has_next"`
	HasPrev   bool `json:"has_prev"`
}

// NewSuccessResponse cria uma resposta de sucesso
func NewSuccessResponse(message string, data interface{}) *BaseResponse {
	return &BaseResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// NewErrorResponse cria uma resposta de erro
func NewErrorResponse(code, message string, status int) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Error:     code,
		Message:   message,
		Status:    status,
		Timestamp: time.Now().Unix(),
	}
}

// NewPaginationResponse cria uma resposta paginada
func NewPaginationResponse(message string, data interface{}, page, limit, total int) *PaginationResponse {
	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginationResponse{
		BaseResponse: BaseResponse{
			Success:   true,
			Message:   message,
			Data:      data,
			Timestamp: time.Now().Unix(),
		},
		Pagination: PaginationInfo{
			Page:      page,
			Limit:     limit,
			Total:     total,
			TotalPage: totalPages,
			HasNext:   page < totalPages,
			HasPrev:   page > 1,
		},
	}
}