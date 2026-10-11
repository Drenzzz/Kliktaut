package services

import (
	"math/rand"
	"regexp"

	"kliktaut-api/internal/config"
)

// Reserved words from §14.1 — also applied to random codes (decision OQ-001:
// a reserved random code is regenerated, not returned as an error).
var ReservedWords = map[string]bool{
	"api": true, "docs": true, "health": true, "admin": true, "login": true,
	"register": true, "static": true, "assets": true, "favicon": true, "robots": true,
}

var aliasRegex = regexp.MustCompile(`^[a-zA-Z0-9-]{3,50}$`)

const base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomSource is injectable so collisions can be tested (TEST-005).
type RandomSource interface {
	Intn(n int) int
}

type randSource struct{ r *rand.Rand }

func (s randSource) Intn(n int) int { return s.r.Intn(n) }

func NewRandomSource() RandomSource {
	return randSource{r: rand.New(rand.NewSource(rand.Int63()))}
}

type Shortener struct {
	cfg    *config.Config
	rand   RandomSource
	exists func(code string) (bool, error)
}

func NewShortener(cfg *config.Config, randSrc RandomSource, exists func(string) (bool, error)) *Shortener {
	return &Shortener{cfg: cfg, rand: randSrc, exists: exists}
}

// Generate produces a random base62 short code; regenerates on collision
// or reserved words, up to 3 attempts.
func (s *Shortener) Generate() (string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		code := s.randomCode()
		if ReservedWords[code] {
			continue
		}
		taken, err := s.exists(code)
		if err != nil {
			return "", err
		}
		if !taken {
			return code, nil
		}
	}
	return "", ErrCodeGenFailed
}

func (s *Shortener) randomCode() string {
	b := make([]byte, s.cfg.ShortCodeLength)
	for i := range b {
		b[i] = base62Chars[s.rand.Intn(len(base62Chars))]
	}
	return string(b)
}

// ValidateAlias validates a custom alias (regex + reserved) and checks
// availability via the exists callback. A valid alias becomes the short_code as-is.
func (s *Shortener) ValidateAlias(alias string) error {
	if !aliasRegex.MatchString(alias) {
		return ErrAliasInvalid
	}
	if ReservedWords[alias] {
		return ErrAliasTaken
	}
	taken, err := s.exists(alias)
	if err != nil {
		return err
	}
	if taken {
		return ErrAliasTaken
	}
	return nil
}
