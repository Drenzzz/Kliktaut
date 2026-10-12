package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"kliktaut-api/internal/models"
)

// LinkCache caches redirect decisions in Redis (§4.2, §9.5).
// Redis errors fail open: callers fall back to PostgreSQL (decision OQ-002).
type LinkCache struct {
	RDB      *redis.Client
	TTLBase  int // CACHE_TTL_SECONDS
}

func NewLinkCache(rdb *redis.Client, ttlBase int) *LinkCache {
	return &LinkCache{RDB: rdb, TTLBase: ttlBase}
}

type entry struct {
	OriginalURL string     `json:"original_url"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

func cacheKey(code string) string { return "link:" + code }

// ttlSeconds = min(TTLBase, expires_at - now); expired → 0 (never cache).
func ttlSeconds(base int, expiresAt *time.Time, now time.Time) int {
	if expiresAt == nil {
		return base
	}
	if !expiresAt.After(now) {
		return 0
	}
	d := int(expiresAt.Sub(now).Seconds()) // truncate sub-second
	if d < base {
		return d
	}
	return base
}

// shouldCache reports whether an active, unexpired link may be cached (§9.5).
func shouldCache(isActive bool, expiresAt *time.Time, now time.Time) bool {
	return isActive && (expiresAt == nil || expiresAt.After(now))
}

// Set stores the link only if it may be cached; silently skips otherwise.
func (c *LinkCache) Set(ctx context.Context, link *models.Link) {
	if !shouldCache(link.IsActive, link.ExpiresAt, time.Now()) {
		return
	}
	ttl := ttlSeconds(c.TTLBase, link.ExpiresAt, time.Now())
	if ttl <= 0 {
		return
	}
	data, err := json.Marshal(entry{
		OriginalURL: link.OriginalURL,
		IsActive:    link.IsActive,
		ExpiresAt:   link.ExpiresAt,
	})
	if err != nil {
		return
	}
	if err := c.RDB.Set(ctx, cacheKey(link.ShortCode), data, time.Duration(ttl)*time.Second).Err(); err != nil {
		log.Printf("cache set error (fail open): %v", err)
	}
}

// Get returns the cached link; miss or Redis error → nil (fall back to DB).
func (c *LinkCache) Get(ctx context.Context, code string) *models.Link {
	data, err := c.RDB.Get(ctx, cacheKey(code)).Bytes()
	if err != nil {
		return nil // includes redis.Nil miss
	}
	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil
	}
	return &models.Link{
		OriginalURL: e.OriginalURL,
		ShortCode:   code,
		IsActive:    e.IsActive,
		ExpiresAt:   e.ExpiresAt,
	}
}

// Delete removes the cache entry (§4.3: after PATCH/DELETE success).
func (c *LinkCache) Delete(ctx context.Context, code string) error {
	if err := c.RDB.Del(ctx, cacheKey(code)).Err(); err != nil {
		log.Printf("cache delete error (fail open): %v", err)
	}
	return nil
}
