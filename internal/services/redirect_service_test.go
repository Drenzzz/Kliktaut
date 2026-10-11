package services

import (
	"testing"
	"time"

	"kliktaut-api/internal/models"
)

func linkPtr(active bool, expires *time.Time) *models.Link {
	return &models.Link{IsActive: active, ExpiresAt: expires, OriginalURL: "https://example.com"}
}

// TEST-006: aturan 404 / 410 inactive / 410 expired
func TestEvaluateRedirect(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name     string
		link     *models.Link
		wantStat int
		wantErr  string // error code yang diharapkan handler, kosong = 302
	}{
		{"nil link → 404", nil, 404, "LINK_NOT_FOUND"},
		{"inactive → 410", linkPtr(false, nil), 410, "LINK_UNAVAILABLE"},
		{"expired → 410", linkPtr(true, &past), 410, "LINK_UNAVAILABLE"},
		{"valid → 302", linkPtr(true, &future), 302, ""},
		{"valid tanpa expiry → 302", linkPtr(true, nil), 302, ""},
	}

	for _, tc := range cases {
		d := EvaluateRedirect(tc.link, now)
		if d.Status != tc.wantStat {
			t.Errorf("%s: status = %d, want %d", tc.name, d.Status, tc.wantStat)
		}
		if tc.wantStat == 302 && d.TargetURL != "https://example.com" {
			t.Errorf("%s: target = %q", tc.name, d.TargetURL)
		}
	}
}

// TEST-006b: tepat pada batas waktu dianggap expired
func TestEvaluateRedirectBoundary(t *testing.T) {
	now := time.Now()
	link := linkPtr(true, &now)
	if d := EvaluateRedirect(link, now); d.Status != 410 {
		t.Errorf("expires_at == now harus 410 (Before tercapai), got %d", d.Status)
	}
}
