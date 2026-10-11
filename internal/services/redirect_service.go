package services

import (
	"time"

	"kliktaut-api/internal/models"
)

// RedirectDecision is the result of evaluating redirect rules §9.2 — pure, easy to test.
type RedirectDecision struct {
	Link      *models.Link
	Status    int // 302 / 404 / 410
	TargetURL string
}

// EvaluateRedirect applies the rules:
//   not found        -> 404
//   is_active=false  -> 410
//   expires_at < now -> 410 (and must not be cached — cache arrives in Phase 3)
//   valid            -> 302 to original_url
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
