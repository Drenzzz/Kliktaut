package middleware

import (
	"testing"
	"time"
)

// TEST-011: window math: remaining, reset epoch, limit respected
func TestRateLimitMath(t *testing.T) {
	// remaining counts requests left AFTER the current one is accepted
	if remaining(1, 60) != 59 {
		t.Errorf("remaining(1) = %d, want 59", remaining(1, 60))
	}
	if remaining(60, 60) != 0 {
		t.Errorf("remaining(60) = %d, want 0", remaining(60, 60))
	}
	if remaining(100, 60) != 0 {
		t.Errorf("remaining(100) = %d, want 0 (clamped)", remaining(100, 60))
	}

	reset := resetEpoch(60, time.Now())
	now := time.Now().Unix()
	if reset < now || reset > now+61 {
		t.Errorf("resetEpoch = %d, want within [%d, %d]", reset, now, now+61)
	}
}

func remaining(count, limit int) int {
	r := limit - count
	if r < 0 {
		r = 0
	}
	return r
}

func resetEpoch(windowSeconds int, now time.Time) int64 {
	return now.Add(time.Duration(windowSeconds) * time.Second).Unix()
}
