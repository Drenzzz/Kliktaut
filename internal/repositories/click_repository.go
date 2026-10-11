package repositories

import (
	"fmt"
	"net"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClickRepository struct {
	DB *gorm.DB
}

func NewClickRepository(db *gorm.DB) *ClickRepository {
	return &ClickRepository{DB: db}
}

// InsertClickAsync versi string ID untuk handler redirect (goroutine).
func (r *ClickRepository) InsertClickAsync(linkID, ip, userAgent, referrer string) error {
	uid, err := uuid.Parse(linkID)
	if err != nil {
		return fmt.Errorf("parse link id: %w", err)
	}
	return r.InsertClick(uid, ip, userAgent, referrer)
}

// InsertClick mencatat click + increment counter atomik dalam satu transaksi.
// Raw SQL: GORM tidak map INET dengan bersih (RISK-007); device/browser/os
// diisi Phase 3 (UA parser).
func (r *ClickRepository) InsertClick(linkID uuid.UUID, ip, userAgent, referrer string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var ipValue interface{}
		if ip != "" {
			ipValue = net.ParseIP(ip)
		}

		if err := tx.Exec(
			`INSERT INTO clicks (link_id, ip_address, user_agent, referrer)
			 VALUES (?, ?::inet, ?, ?)`,
			linkID, ipValue, userAgent, referrer,
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
