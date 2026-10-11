package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"kliktaut-api/internal/services"
	"kliktaut-api/internal/utils"
)

type AnalyticsHandler struct {
	svc *services.AnalyticsService
}

func NewAnalyticsHandler(svc *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// GET /api/v1/links/:id/analytics
func (h *AnalyticsHandler) LinkAnalytics(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid link ID")
	}

	if h.ensureLinkExists(c, id) {
		return nil // error response already written
	}

	result, err := h.svc.ForLink(id)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected server error occurred")
	}

	return utils.Success(c, fiber.StatusOK, result)
}

// GET /api/v1/links/:id/clicks
func (h *AnalyticsHandler) ClicksList(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid link ID")
	}

	if h.ensureLinkExists(c, id) {
		return nil // error response already written
	}

	page := fiber.Query[int](c, "page", 1)
	perPage := fiber.Query[int](c, "per_page", 20)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	clicks, total, err := h.svc.ClicksList(id, page, perPage)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected server error occurred")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, clicks, utils.MetaInfo{
		Page: page, PerPage: perPage, Total: total,
	})
}

// GET /api/v1/stats
func (h *AnalyticsHandler) Overview(c fiber.Ctx) error {
	stats, err := h.svc.Overview()
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected server error occurred")
	}
	return utils.Success(c, fiber.StatusOK, stats)
}

// ensureLinkExists writes the error response itself and reports whether
// the request was handled (link missing/invalid) — utils.Error returns nil
// on success, so propagating its value would silently drop the 404.
func (h *AnalyticsHandler) ensureLinkExists(c fiber.Ctx, id string) bool {
	exists, err := h.svc.ForLinkExist(id)
	if err != nil {
		if errors.Is(err, services.ErrLinkNotFound) {
			_ = utils.Error(c, fiber.StatusNotFound, "LINK_NOT_FOUND", "Link not found")
			return true
		}
		_ = utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected server error occurred")
		return true
	}
	_ = exists
	return false
}
