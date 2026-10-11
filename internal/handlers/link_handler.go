package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

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
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Body tidak valid")
	}

	// validator struct tag + cek manual scheme (validator `url` menerima ftp://)
	if strings.HasPrefix(req.OriginalURL, "ftp://") || strings.HasPrefix(req.OriginalURL, "file://") {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "URL harus http/https")
	}
	if req.CustomAlias != "" && (len(req.CustomAlias) < 3 || len(req.CustomAlias) > 50) {
		return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "custom_alias harus 3-50 karakter")
	}

	link, err := h.svc.Create(req.OriginalURL, req.CustomAlias, req.Title, req.ExpiresAt)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAliasTaken):
			return utils.Error(c, fiber.StatusConflict, "ALIAS_TAK_TERSEDIA", "Custom alias sudah dipakai atau tidak diizinkan")
		case errors.Is(err, services.ErrAliasInvalid):
			return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Custom alias hanya boleh huruf, angka, dan dash (3-50 karakter)")
		case errors.Is(err, services.ErrInvalidRequest):
			return utils.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "original_url harus http/https")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan di server")
		}
	}

	return utils.Success(c, fiber.StatusCreated, toLinkResponse(link, h.svc.ShortURL(link.ShortCode)))
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
