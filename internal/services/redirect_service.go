package services

import (
	"time"

	"kliktaut-api/internal/models"
)

// RedirectDecision hasil evaluasi aturan redirect §9.2 — murni, mudah dites.
type RedirectDecision struct {
	Link      *models.Link
	Status    int // 302 / 404 / 410
	TargetURL string
}

// EvaluateRedirect menerapkan aturan:
//   tidak ada        → 404
//   is_active=false  → 410
//   expires_at < now → 410 (dan jangan di-cache — cache baru Phase 3)
//   valid            → 302 ke original_url
func EvaluateRedirect(link *models.Link, now time.Time) RedirectDecision {
	if link == nil {
		return RedirectDecision{Status: 404}
	}
	if !link.IsActive {
		return RedirectDecision{Link: link, Status: 410}
	}
	if link.ExpiresAt != nil && !link.ExpiresAt.After(now) {
		return RedirectDecision{Link: link, Status: 410}
	}
	return RedirectDecision{Link: link, Status: 302, TargetURL: link.OriginalURL}
}
