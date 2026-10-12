package middleware

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"kliktaut-api/internal/utils"
)

// RateLimit is a Redis fixed-window limiter (§9.4). Redis errors fail open
// (decision OQ-002) so availability wins at portfolio scale.
func RateLimit(rdb *redis.Client, prefix string, limit int, windowSeconds int) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		apiKey, _ := c.Locals("api_key").(string)
		identity := apiKey
		if identity == "" {
			identity = c.IP()
		}
		key := fmt.Sprintf("rl:%s:%s", prefix, identity)
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("rate limit redis error (fail open): %v", err)
			return c.Next()
		}
		if count == 1 {
			rdb.Expire(ctx, key, time.Duration(windowSeconds)*time.Second)
		}

		remaining := limit - int(count)
		if remaining < 0 {
			remaining = 0
		}
		reset := time.Now().Add(time.Duration(windowSeconds) * time.Second).Unix()

		c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))

		if int(count) > limit {
			return utils.Error(c, fiber.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
		}
		return c.Next()
	}
}
