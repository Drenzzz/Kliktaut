package cache

import (
	"testing"
	"time"
)

// TEST-012: TTL = min(CACHE_TTL, expires_at - now); null → full TTL;
// expired/inactive links are never cached (§9.5).
func TestTTLCalculation(t *testing.T) {
	now := time.Now()
	full := 3600

	if got := ttlSeconds(full, nil, now); got != full {
		t.Errorf("null expiry: ttl = %d, want %d", got, full)
	}

	future := now.Add(10 * time.Minute)
	if got := ttlSeconds(full, &future, now); got < 599 || got > 600 {
		t.Errorf("expiry in 10m: ttl = %d, want 599-600", got)
	}

	sooner := now.Add(30 * time.Second)
	if got := ttlSeconds(full, &sooner, now); got < 29 || got > 30 {
		t.Errorf("expiry in 30s: ttl = %d, want 29-30", got)
	}

	past := now.Add(-time.Hour)
	if got := ttlSeconds(full, &past, now); got != 0 {
		t.Errorf("expired: ttl = %d, want 0 (do not cache)", got)
	}
}

func TestShouldCache(t *testing.T) {
	now := time.Now()
	if shouldCache(true, nil, now) != true {
		t.Error("active + null expiry: should cache")
	}
	if shouldCache(false, nil, now) != false {
		t.Error("inactive: must not cache")
	}
	past := now.Add(-time.Minute)
	if shouldCache(true, &past, now) != false {
		t.Error("expired: must not cache")
	}
	future := now.Add(time.Hour)
	if shouldCache(true, &future, now) != true {
		t.Error("active + future expiry: should cache")
	}
}
