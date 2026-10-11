package services

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"kliktaut-api/internal/config"
	"kliktaut-api/internal/models"
	"kliktaut-api/internal/repositories"
)

var ErrAliasTaken = errors.New("ALIAS_TAK_TERSEDIA")
var ErrAliasInvalid = errors.New("ALIAS_TIDAK_VALID")
var ErrLinkNotFound = errors.New("LINK_NOT_FOUND")
var ErrInvalidRequest = errors.New("INVALID_REQUEST")
var ErrCodeGenFailed = errors.New("GAGAL_GENERATE_KODE")

type LinkService struct {
	cfg        *config.Config
	links      *repositories.LinkRepository
	shortener  *Shortener
}

func NewLinkService(cfg *config.Config, links *repositories.LinkRepository, s *Shortener) *LinkService {
	return &LinkService{cfg: cfg, links: links, shortener: s}
}

// Create membuat link: alias valid menjadi short_code, selain itu kode random.
func (svc *LinkService) Create(originalURL, alias, title string, expiresAt *time.Time) (*models.Link, error) {
	originalURL = strings.TrimSpace(originalURL)
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		return nil, ErrInvalidRequest
	}

	var code string
	if alias != "" {
		if err := svc.shortener.ValidateAlias(alias); err != nil {
			return nil, err
		}
		code = alias
	} else {
		generated, err := svc.shortener.Generate()
		if err != nil {
			return nil, err
		}
		code = generated
	}

	link := &models.Link{
		OriginalURL: originalURL,
		ShortCode:   code,
		ExpiresAt:   expiresAt,
	}
	if title != "" {
		t := title
		link.Title = &t
	}

	if err := svc.links.Create(link); err != nil {
		// Kode random bisa collision di DB walau exists-check lolos (race);
		// kalau ini custom alias, biarkan di-map jadi alias taken.
		return nil, err
	}
	return link, nil
}

func (svc *LinkService) ShortURL(code string) string {
	return svc.cfg.BaseURL + "/" + code
}

// List membungkus repo untuk handler (pagination + search).
func (svc *LinkService) List(page, perPage int, search string) ([]models.Link, int64, error) {
	return svc.links.List(page, perPage, search)
}

// GetByID mengembalikan link atau ErrLinkNotFound.
func (svc *LinkService) GetByID(id string) (*models.Link, error) {
	link, err := svc.links.GetByID(id)
	if err != nil {
		// GORM membungkus ErrRecordNotFound tanpa Unwrap — cek langsung
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}
	return link, nil
}
