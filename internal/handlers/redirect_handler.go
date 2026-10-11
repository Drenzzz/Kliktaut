package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v3"

	"kliktaut-api/internal/repositories"
	"kliktaut-api/internal/services"
	"kliktaut-api/internal/utils"
)

type RedirectHandler struct {
	svc     *services.LinkService
	clicks  *repositories.ClickRepository
}

func NewRedirectHandler(svc *services.LinkService, clicks *repositories.ClickRepository) *RedirectHandler {
	return &RedirectHandler{svc: svc, clicks: clicks}
}

// GET /:shortCode — jalur paling kritikal: redirect + catat click async.
// Click tracking must never block the response (§4.2).
func (h *RedirectHandler) Redirect(c fiber.Ctx) error {
	code := c.Params("shortCode")

	link, err := h.svc.GetByShortCode(code)
	if err != nil {
		return utils.Error(c, fiber.StatusNotFound, "LINK_NOT_FOUND", "Short link not found")
	}

	decision := services.EvaluateRedirect(link, time.Now())
	switch decision.Status {
	case 404:
		return utils.Error(c, fiber.StatusNotFound, "LINK_NOT_FOUND", "Short link not found")
	case 410:
		return utils.Error(c, fiber.StatusGone, "LINK_UNAVAILABLE", "Short link is no longer available")
	}

	// async: INSERT click + increment the counter in a separate goroutine;
	// errors are logged, not retried (decision OQ-003 — loss accepted in v1.0)
	go func(linkID, ip, ua, referrer string) {
		if err := h.clicks.InsertClickAsync(linkID, ip, ua, referrer); err != nil {
			log.Printf("click tracking error (link %s): %v", linkID, err)
		}
	}(link.ID.String(), c.IP(), c.Get("User-Agent"), c.Get("Referer"))

	c.Set("Location", decision.TargetURL)
	return c.SendStatus(fiber.StatusFound) // 302, not 301 (§9.2)
}
