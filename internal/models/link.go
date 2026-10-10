package models

import (
	"time"

	"github.com/google/uuid"
)

type Link struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OriginalURL string     `gorm:"column:original_url;not null" json:"original_url"`
	ShortCode   string     `gorm:"column:short_code;uniqueIndex;size:50;not null" json:"short_code"`
	Title       *string    `gorm:"size:255" json:"title"`
	IsActive    bool       `gorm:"column:is_active;not null;default:true" json:"is_active"`
	ExpiresAt   *time.Time `gorm:"column:expires_at" json:"expires_at"`
	ClickCount  int64      `gorm:"column:click_count;not null;default:0" json:"click_count"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (Link) TableName() string { return "links" }
