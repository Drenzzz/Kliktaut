package utils

import (
	"regexp"
	"strings"

	uap "github.com/ua-parser/uap-go/uaparser"
)

// Bot patterns §9.3, case-insensitive (decision OQ-012) plus a few
// common link-preview bots.
var botRegex = regexp.MustCompile(`(?i)bot|crawler|spider|curl|wget|python-requests|slurp|bingpreview|facebookexternalhit|preview`)

var uaParser = mustParser()

// Parser uses the embedded default regex definitions.
func mustParser() *uap.Parser {
	p := uap.NewFromSaved()
	if p == nil {
		panic("uap-go: failed to build parser from saved definitions")
	}
	return p
}

// UAInfo holds parsed user-agent fields stored in clicks.device_type etc.
type UAInfo struct {
	DeviceType string // mobile | desktop | tablet | bot (§5.4)
	Browser    string
	OS         string
}

// ParseUA parses a user agent string. Unknown/empty values fall back to
// desktop / Other so analytics never stores empty strings.
func ParseUA(ua string) UAInfo {
	info := UAInfo{DeviceType: "desktop", Browser: "Other", OS: "Other"}
	if strings.TrimSpace(ua) == "" {
		return info
	}

	c := uaParser.Parse(ua)

	// Device family: uap-go emits "Generic Smartphone"/"Generic Tablet"
	// or brand names like iPad. Fall back to OS family for phones.
	dev := ""
	os := ""
	browser := ""
	if c.Device != nil {
		dev = c.Device.Family
	}
	if c.Os != nil {
		os = c.Os.Family
	}
	if c.UserAgent != nil {
		browser = c.UserAgent.Family
	}

	switch {
	case strings.Contains(strings.ToLower(dev), "tablet") || strings.EqualFold(dev, "iPad"):
		info.DeviceType = "tablet"
	case dev != "" && dev != "Other" && dev != "Generic Smartphone":
		// Brand device (e.g. "Pixel 8") with no tablet marker: phones and
		// desktops both land here; OS family disambiguates.
		osLower := strings.ToLower(os)
		if strings.Contains(osLower, "ios") || strings.Contains(osLower, "android") || strings.Contains(osLower, "windows phone") {
			info.DeviceType = "mobile"
		}
	case dev == "Generic Smartphone":
		info.DeviceType = "mobile"
	}

	// Mobile Safari marker covers iOS devices uap-go reports without a family.
	if strings.Contains(ua, "iPad") {
		info.DeviceType = "tablet"
	} else if info.DeviceType == "desktop" && strings.Contains(ua, "Mobile") {
		info.DeviceType = "mobile"
	}

	if browser != "" && browser != "Other" {
		info.Browser = browser
	}
	if os != "" && os != "Other" {
		info.OS = os
	}

	return info
}

// IsBot reports whether the user agent matches the bot patterns.
func IsBot(ua string) bool {
	return botRegex.MatchString(ua)
}
