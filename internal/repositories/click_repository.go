package repositories

import (
	"fmt"
	"net"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kliktaut-api/internal/utils"
)

type ClickRepository struct {
	DB *gorm.DB
}

func NewClickRepository(db *gorm.DB) *ClickRepository {
	return &ClickRepository{DB: db}
}

// InsertClickAsync takes a string ID for the redirect handler goroutine.
func (r *ClickRepository) InsertClickAsync(linkID, ip, userAgent, referrer string) error {
	uid, err := uuid.Parse(linkID)
	if err != nil {
		return fmt.Errorf("parse link id: %w", err)
	}
	return r.InsertClick(uid, ip, userAgent, referrer)
}

// InsertClick records a click and atomically increments the counter in one
// transaction. Raw SQL: GORM does not map INET cleanly (RISK-007).
// UA parsing happens here so device/browser/os land in the same row (§9.3).
func (r *ClickRepository) InsertClick(linkID uuid.UUID, ip, userAgent, referrer string) error {
	device, browser, os := parseFields(userAgent)

	return r.DB.Transaction(func(tx *gorm.DB) error {
		var ipValue interface{}
		if ip != "" {
			ipValue = net.ParseIP(ip)
		}

		if err := tx.Exec(
			`INSERT INTO clicks (link_id, ip_address, user_agent, referrer, device_type, browser, os)
			 VALUES (?, ?::inet, ?, ?, ?, ?, ?)`,
			linkID, ipValue, userAgent, referrer, device, browser, os,
		).Error; err != nil {
			return fmt.Errorf("insert click: %w", err)
		}

		if err := tx.Exec(
			`UPDATE links SET click_count = click_count + 1 WHERE id = ?`,
			linkID,
		).Error; err != nil {
			return fmt.Errorf("increment click_count: %w", err)
		}

		return nil
	})
}

func parseFields(userAgent string) (device, browser, os string) {
	if utils.IsBot(userAgent) {
		return "bot", userAgent, userAgent
	}
	info := utils.ParseUA(userAgent)
	return info.DeviceType, info.Browser, info.OS
}
