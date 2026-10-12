package middleware

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// TEST-010: valid / invalid / missing API key
func TestAPIKey(t *testing.T) {
	app := fiber.New()
	app.Use(APIKey("secret-key-123"))
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("ok") })

	cases := []struct {
		name   string
		key    string
		want   int
	}{
		{"valid key", "secret-key-123", 200},
		{"wrong key", "wrong", 401},
		{"empty key", "", 401},
		{"no header", "skip", 401}, // special: header not set below
	}

	for _, tc := range cases {
		req, _ := http.NewRequest("GET", "/ping", nil)
		if tc.key != "skip" {
			req.Header.Set("X-API-Key", tc.key)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if resp.StatusCode != tc.want {
			t.Errorf("%s: status = %d, want %d", tc.name, resp.StatusCode, tc.want)
		}
	}
}
