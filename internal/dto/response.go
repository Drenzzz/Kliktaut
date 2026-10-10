package dto

import (
	"time"

	"github.com/google/uuid"
)

// ===== Link Responses =====

type LinkResponse struct {
	ID          uuid.UUID  `json:"id"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	Title       *string    `json:"title"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
	ClickCount  int64      `json:"click_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type LinkListResponse struct {
	ID         uuid.UUID `json:"id"`
	ShortCode  string    `json:"short_code"`
	ShortURL   string    `json:"short_url"`
	Title      *string   `json:"title"`
	IsActive   bool      `json:"is_active"`
	ExpiresAt  *time.Time `json:"expires_at"`
	ClickCount int64     `json:"click_count"`
	CreatedAt  time.Time `json:"created_at"`
}

type Meta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}
