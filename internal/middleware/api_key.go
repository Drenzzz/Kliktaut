package middleware

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v3"

	"kliktaut-api/internal/utils"
)

// APIKey protects /api/v1/* with a static key from env (REQ-024).
// Comparison is constant-time to prevent timing attacks.
func APIKey(key string) fiber.Handler {
	return func(c fiber.Ctx) error {
		provided := c.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(key)) != 1 {
			return utils.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid API key")
		}
		c.Locals("api_key", provided)
		return c.Next()
	}
}
