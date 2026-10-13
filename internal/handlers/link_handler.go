package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"kliktaut-api/internal/dto"
	"kliktaut-api/internal/models"
	"kliktaut-api/internal/services"
	"kliktaut-api/internal/utils"
)

type LinkHandler struct {
	svc *services.LinkService
}

func NewLinkHandler(svc *services.LinkService) *LinkHandler {
	return &LinkHandler{svc: svc}
}

// POST /api/v1/links
func (h *LinkHandler) Create(c fiber.Ctx) error {
	var req dto.CreateLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// strict validation §10.2: scheme http/https, max 2048, alias format
	if err := utils.ValidURL(req.OriginalURL); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	link, err := h.svc.Create(req.OriginalURL, req.CustomAlias, req.Title, req.ExpiresAt)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAliasTaken):
			return utils.Error(c, fiber.StatusConflict, "ALIAS_TAKEN", "Custom alias is already in use or not allowed")
		case errors.Is(err, services.ErrAliasInvalid):
			return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Custom alias may only contain letters, numbers, and dashes (3-50 characters)")
		case errors.Is(err, services.ErrInvalidRequest):
			return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "original_url must use http or https")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected server error occurred")
		}
	}

	return utils.Success(c, fiber.StatusCreated, toLinkResponse(link, h.svc.ShortURL(link.ShortCode)))
}

// GET /api/v1/links?page=&per_page=&search=
func (h *LinkHandler) List(c fiber.Ctx) error {
	page := fiber.Query[int](c, "page", 1)
	perPage := fiber.Query[int](c, "per_page", 20)
	search := fiber.Query[string](c, "search", "")

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	if len(search) > 50 {
		search = search[:50]
	}

	links, total, err := h.svc.List(page, perPage, search)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected server error occurred")
	}

	data := make([]dto.LinkListResponse, 0, len(links))
	for i := range links {
		l := &links[i]
		data = append(data, dto.LinkListResponse{
			ID:         l.ID,
			ShortCode:  l.ShortCode,
			ShortURL:   h.svc.ShortURL(l.ShortCode),
			Title:      l.Title,
			IsActive:   l.IsActive,
			ExpiresAt:  l.ExpiresAt,
			ClickCount: l.ClickCount,
			CreatedAt:  l.CreatedAt,
		})
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, data, utils.MetaInfo{
		Page: page, PerPage: perPage, Total: total,
	})
}

// GET /api/v1/links/:id
func (h *LinkHandler) GetByID(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid link ID")
	}

	link, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, services.ErrLinkNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.Error(c, fiber.StatusNotFound, "LINK_NOT_FOUND", "Link not found")
		}
		return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan di server")
	}

	return utils.Success(c, fiber.StatusOK, toLinkResponse(link, h.svc.ShortURL(link.ShortCode)))
}

// PATCH /api/v1/links/:id: minimal 1 field; short_code immutable.
func (h *LinkHandler) Update(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid link ID")
	}

	link, err := h.svc.Update(id, c.Body())
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRequest):
			return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "At least one field is required; only original_url/title/expires_at/is_active can be updated")
		case errors.Is(err, services.ErrLinkNotFound):
			return utils.Error(c, fiber.StatusNotFound, "LINK_NOT_FOUND", "Link not found")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected server error occurred")
		}
	}

	return utils.Success(c, fiber.StatusOK, toLinkResponse(link, h.svc.ShortURL(link.ShortCode)))
}

// DELETE /api/v1/links/:id: soft delete (is_active=false).
func (h *LinkHandler) Delete(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid link ID")
	}

	if err := h.svc.SoftDelete(id); err != nil {
		if errors.Is(err, services.ErrLinkNotFound) {
			return utils.Error(c, fiber.StatusNotFound, "LINK_NOT_FOUND", "Link not found")
		}
		return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected server error occurred")
	}

	return utils.Success(c, fiber.StatusOK, fiber.Map{"message": "Link deactivated successfully"})
}

func toLinkResponse(link *models.Link, shortURL string) dto.LinkResponse {
	return dto.LinkResponse{
		ID:          link.ID,
		OriginalURL: link.OriginalURL,
		ShortCode:   link.ShortCode,
		ShortURL:    shortURL,
		Title:       link.Title,
		IsActive:    link.IsActive,
		ExpiresAt:   link.ExpiresAt,
		ClickCount:  link.ClickCount,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   &link.UpdatedAt,
	}
}
