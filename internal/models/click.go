package models

import (
	"net"
	"time"

	"github.com/google/uuid"
)

type Click struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LinkID     uuid.UUID `gorm:"type:uuid;not null" json:"link_id"`
	IPAddress  *net.IPNet
	UserAgent  *string   `gorm:"column:user_agent" json:"user_agent"`
	Referrer   *string   `gorm:"column:referrer" json:"referrer"`
	Country    *string   `gorm:"size:10" json:"country"`
	DeviceType *string   `gorm:"column:device_type;size:30" json:"device_type"`
	Browser    *string   `gorm:"size:100" json:"browser"`
	OS         *string   `gorm:"column:os;size:100" json:"os"`
	ClickedAt  time.Time `gorm:"column:clicked_at;not null" json:"clicked_at"`
}

func (Click) TableName() string { return "clicks" }
