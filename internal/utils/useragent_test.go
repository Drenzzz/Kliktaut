package utils

import "testing"

// TEST-008: device/browser/os parsing
func TestParseUA(t *testing.T) {
	cases := []struct {
		name   string
		ua     string
		device string
		brow   string
		os     string
	}{
		{
			"chrome desktop",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"desktop", "Chrome", "Windows",
		},
		{
			"android mobile",
			"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			"mobile", "Chrome Mobile", "Android",
		},
		{
			"iphone safari",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			"mobile", "Mobile Safari", "iOS",
		},
		{
			"tablet ipad",
			"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			"tablet", "Mobile Safari", "iOS",
		},
		{
			"empty ua",
			"",
			"desktop", "Other", "Other",
		},
	}

	for _, tc := range cases {
		got := ParseUA(tc.ua)
		if got.DeviceType != tc.device {
			t.Errorf("%s: device = %q, want %q", tc.name, got.DeviceType, tc.device)
		}
		if got.Browser == "" || got.OS == "" {
			t.Errorf("%s: browser/os kosong (%q/%q)", tc.name, got.Browser, got.OS)
		}
	}
}

// TEST-009: bot detection case-insensitive
func TestIsBot(t *testing.T) {
	bots := []string{
		"curl/8.5.0",
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"Wget/1.21.2",
		"python-requests/2.31.0",
		"GPTBot/1.0",
		"BingPreview/1.0b",
		"Mozilla/5.0 (compatible; bingbot/2.0)",
		"facebookexternalhit/1.1",
	}
	for _, ua := range bots {
		if !IsBot(ua) {
			t.Errorf("IsBot(%q) = false, want true", ua)
		}
	}

	humans := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0) Safari/604.1",
		"",
	}
	for _, ua := range humans {
		if IsBot(ua) {
			t.Errorf("IsBot(%q) = true, want false", ua)
		}
	}
}
