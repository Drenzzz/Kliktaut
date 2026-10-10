package dto

import "time"

// ===== Requests =====

type CreateLinkRequest struct {
	OriginalURL  string     `json:"original_url" validate:"required,url,max=2048"`
	CustomAlias  string     `json:"custom_alias" validate:"omitempty,min=3,max=50"`
	Title        string     `json:"title" validate:"omitempty,max=255"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

type UpdateLinkRequest struct {
	OriginalURL *string    `json:"original_url" validate:"omitempty,url,max=2048"`
	Title       *string    `json:"title" validate:"omitempty,max=255"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    *bool      `json:"is_active"`
}
