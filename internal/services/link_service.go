package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"kliktaut-api/internal/cache"
	"kliktaut-api/internal/config"
	"kliktaut-api/internal/models"
	"kliktaut-api/internal/repositories"
)

var ErrAliasTaken = errors.New("ALIAS_TAKEN")
var ErrAliasInvalid = errors.New("ALIAS_INVALID")
var ErrLinkNotFound = errors.New("LINK_NOT_FOUND")
var ErrInvalidRequest = errors.New("INVALID_REQUEST")
var ErrCodeGenFailed = errors.New("CODE_GENERATION_FAILED")

type LinkService struct {
	cfg   *config.Config
	links *repositories.LinkRepository
	short *Shortener
	cache *cache.LinkCache
}

func NewLinkService(cfg *config.Config, links *repositories.LinkRepository, s *Shortener, c *cache.LinkCache) *LinkService {
	return &LinkService{cfg: cfg, links: links, short: s, cache: c}
}

// Create builds a link: a valid alias becomes the short_code, otherwise a random code.
func (svc *LinkService) Create(originalURL, alias, title string, expiresAt *time.Time) (*models.Link, error) {
	originalURL = strings.TrimSpace(originalURL)
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		return nil, ErrInvalidRequest
	}

	var code string
	if alias != "" {
		if err := svc.short.ValidateAlias(alias); err != nil {
			return nil, err
		}
		code = alias
	} else {
		generated, err := svc.short.Generate()
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
		// A random code can still collide in the DB even if the exists-check passed (race);
		// for custom aliases, surface it as alias taken.
		return nil, err
	}
	return link, nil
}

func (svc *LinkService) ShortURL(code string) string {
	return svc.cfg.BaseURL + "/" + code
}

// List wraps the repository for handlers (pagination + search).
func (svc *LinkService) List(page, perPage int, search string) ([]models.Link, int64, error) {
	return svc.links.List(page, perPage, search)
}

// GetByID returns a link or ErrLinkNotFound.
func (svc *LinkService) GetByID(id string) (*models.Link, error) {
	link, err := svc.links.GetByID(id)
	if err != nil {
		// GORM wraps ErrRecordNotFound without Unwrap, so compare directly
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}
	return link, nil
}

// Update patches a subset of fields; short_code is immutable; at least one field required.
// Presence check uses a map because *time.Time cannot distinguish
// "expires_at: null" (clear expiry) from "field not sent" (§6.4).
func (svc *LinkService) Update(id string, body []byte) (*models.Link, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, ErrInvalidRequest
	}
	delete(raw, "short_code") // immutable, ignored
	delete(raw, "id")         // immutable

	if len(raw) == 0 {
		return nil, ErrInvalidRequest
	}

	fields := make(map[string]interface{}, 4)

	if v, ok := raw["original_url"]; ok {
		var u string
		if err := json.Unmarshal(v, &u); err != nil {
			return nil, ErrInvalidRequest
		}
		u = strings.TrimSpace(u)
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return nil, ErrInvalidRequest
		}
		fields["original_url"] = u
	}
	if v, ok := raw["title"]; ok {
		var t string
		if err := json.Unmarshal(v, &t); err != nil {
			return nil, ErrInvalidRequest
		}
		if len(t) > 255 {
			return nil, ErrInvalidRequest
		}
		fields["title"] = t
	}
	if v, ok := raw["expires_at"]; ok {
		var t *time.Time
		if err := json.Unmarshal(v, &t); err != nil {
			return nil, ErrInvalidRequest
		}
		fields["expires_at"] = t // null -> NULL in DB (clear expiry)
	}
	if v, ok := raw["is_active"]; ok {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return nil, ErrInvalidRequest
		}
		fields["is_active"] = b
	}

	if len(fields) == 0 {
		return nil, ErrInvalidRequest
	}

	if err := svc.links.UpdateFields(id, fields); err != nil {
		return nil, err
	}
	link, err := svc.GetByID(id)
	if err != nil {
		return nil, err
	}
	// §4.3: any successful PATCH invalidates the redirect cache
	svc.invalidateCache(link.ShortCode)
	return link, nil
}

// GetByShortCode for the redirect handler.
func (svc *LinkService) GetByShortCode(code string) (*models.Link, error) {
	link, err := svc.links.GetByShortCode(code)
	if err != nil {
		// GORM wraps ErrRecordNotFound without Unwrap, so compare directly
		if err == gorm.ErrRecordNotFound {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}
	return link, nil
}

// SoftDelete deactivates a link without removing the row (REQ-011).
func (svc *LinkService) SoftDelete(id string) error {
	link, err := svc.GetByID(id)
	if err != nil {
		return err
	}
	if err := svc.links.UpdateFields(id, map[string]interface{}{"is_active": false}); err != nil {
		return err
	}
	// §4.3: DELETE must stop redirects within milliseconds, not after TTL
	svc.invalidateCache(link.ShortCode)
	return nil
}

func (svc *LinkService) invalidateCache(code string) {
	if svc.cache != nil {
		if err := svc.cache.Delete(context.Background(), code); err != nil {
			log.Printf("cache invalidation error (code %s): %v", code, err)
		}
	}
}
