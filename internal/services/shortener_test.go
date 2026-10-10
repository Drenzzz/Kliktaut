package services

import (
	"errors"
	"strings"
	"testing"

	"kliktaut-api/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{ShortCodeLength: 7}
}

// TEST-003: panjang & karakter base62 benar
func TestGenerateBase62(t *testing.T) {
	s := NewShortener(testConfig(), NewRandomSource(), func(string) (bool, error) { return false, nil })

	for i := 0; i < 200; i++ {
		code, err := s.Generate()
		if err != nil {
			t.Fatalf("Generate() error: %v", err)
		}
		if len(code) != 7 {
			t.Fatalf("panjang kode = %d, want 7", len(code))
		}
		for _, ch := range code {
			if !strings.ContainsRune(base62Chars, ch) {
				t.Fatalf("karakter %q di luar base62", ch)
			}
		}
	}
}

// TEST-004: alias valid, invalid, dan reserved
func TestValidateAlias(t *testing.T) {
	exists := func(code string) (bool, error) { return code == "sudah-dipakai", nil }
	s := NewShortener(testConfig(), NewRandomSource(), exists)

	cases := []struct {
		alias string
		want  error
	}{
		{"produk-baru", nil},               // valid
		{"ab", ErrAliasInvalid},            // < 3 char
		{strings.Repeat("a", 51), ErrAliasInvalid}, // > 50 char
		{"ada spasi", ErrAliasInvalid},     // karakter dilarang
		{"underscore_x", ErrAliasInvalid},  // karakter dilarang
		{"api", ErrAliasTaken},             // reserved
		{"favicon", ErrAliasTaken},         // reserved
		{"sudah-dipakai", ErrAliasTaken},   // sudah ada
	}

	for _, tc := range cases {
		err := s.ValidateAlias(tc.alias)
		if !errors.Is(err, tc.want) {
			t.Errorf("ValidateAlias(%q) = %v, want %v", tc.alias, err, tc.want)
		}
	}
}

// TEST-005: collision → regenerate maks 3x lalu error
func TestGenerateCollision(t *testing.T) {
	attempts := 0
	alwaysTaken := func(string) (bool, error) { attempts++; return true, nil }
	s := NewShortener(testConfig(), NewRandomSource(), alwaysTaken)

	if _, err := s.Generate(); !errors.Is(err, ErrCodeGenFailed) {
		t.Fatalf("want ErrCodeGenFailed, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("percobaan = %d, want 3", attempts)
	}
}

// TEST-005b: kode random yang reserved di-regenerate, bukan error
func TestGenerateReservedRegenerated(t *testing.T) {
	// source yang selalu menghasilkan 'f','a','v','i','c','o','n' lalu kode biasa
	seq := []int{5, 0, 21, 8, 2, 14, 13, 0, 0, 0, 0, 0, 0, 0}
	idx := 0
	fixed := fixedSource{seq: seq, idx: &idx}
	seen := 0
	s := NewShortener(testConfig(), fixed, func(code string) (bool, error) {
		seen++
		return false, nil
	})

	code, err := s.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if code == "favicon" {
		t.Fatal("reserved word tidak boleh dikembalikan")
	}
	if len(code) != 7 {
		t.Fatalf("kode hasil regenerate salah: %q", code)
	}
}

type fixedSource struct {
	seq []int
	idx *int
}

func (f fixedSource) Intn(n int) int {
	i := *f.idx
	*f.idx++
	if i >= len(f.seq) {
		return 0
	}
	return f.seq[i] % n
}
