// Seed inserts demo links and clicks so analytics look alive.
// Idempotent: skips when the demo links already exist.
package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"

	"gorm.io/gorm/logger"

	"kliktaut-api/internal/config"
	"kliktaut-api/internal/database"
)

type seedLink struct {
	code, url, title string
}

var demoLinks = []seedLink{
	{"kliktaut-docs", "https://go.dev/doc/", "Go Documentation"},
	{"kliktaut-fiber", "https://docs.gofiber.io/", "Fiber Web Framework"},
	{"kliktaut-gorm", "https://gorm.io/docs/", "GORM Documentation"},
	{"kliktaut-redis", "https://redis.io/docs/latest/", "Redis Docs"},
	{"kliktaut-scalar", "https://github.com/scalar/scalar", "Scalar API Reference"},
}

var uas = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) Chrome/120.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) Mobile Safari/604.1",
	"curl/8.5.0", // bot row for analytics bot-exclusion checks
}

var referrers = []*string{nil, strPtr("https://twitter.com"), strPtr("https://news.ycombinator.com"), strPtr("https://github.com")}

var browsers = []string{"Chrome", "Safari", "Chrome Mobile", "Mobile Safari", "Firefox"}

var rnd = rand.New(rand.NewSource(42))

func strPtr(s string) *string { return &s }

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	db.Logger = db.Logger.LogMode(logger.Silent)

	// idempotency check: skip when the demo set is present
	var count int64
	if err := db.Table("links").Where("short_code LIKE ?", "kliktaut-%").Count(&count).Error; err != nil {
		log.Fatalf("count links: %v", err)
	}
	if count > 0 {
		fmt.Println("seed: demo links already present, nothing to do")
		return
	}


	now := time.Now()
	for i, l := range demoLinks {
		created := now.AddDate(0, 0, -(10 - i))
		res := db.Exec(
			`INSERT INTO links (original_url, short_code, title, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?)`,
			l.url, l.code, l.title, created, created,
		)
		if res.Error != nil {
			log.Fatalf("insert link %s: %v", l.code, res.Error)
		}

		var linkID string
		if err := db.Table("links").Select("id").Where("short_code = ?", l.code).Scan(&linkID).Error; err != nil {
			log.Fatalf("find link %s: %v", l.code, err)
		}

		clicks := 8 + rnd.Intn(15)
		for j := 0; j < clicks; j++ {
			clicked := now.Add(-time.Duration(rnd.Intn(7*24)) * time.Hour)
			ua := uas[rnd.Intn(len(uas))]
			device, browser, os := classify(ua)
			var ip interface{}
			if rnd.Intn(4) > 0 { // some clicks without IP
				ip = net.ParseIP(fmt.Sprintf("10.0.%d.%d", rnd.Intn(255), rnd.Intn(255)))
			}
			if err := db.Exec(
				`INSERT INTO clicks (link_id, ip_address, user_agent, referrer, device_type, browser, os, clicked_at)
				 VALUES (?, ?::inet, ?, ?, ?, ?, ?, ?)`,
				linkID, ip, ua, referrers[rnd.Intn(len(referrers))], device, browser, os, clicked,
			).Error; err != nil {
				log.Fatalf("insert click: %v", err)
			}
		}

		// counter must match the inserted rows
		if err := db.Exec(
			`UPDATE links SET click_count = (SELECT COUNT(*) FROM clicks WHERE link_id = ?) WHERE id = ?`,
			linkID, linkID,
		).Error; err != nil {
			log.Fatalf("sync counter %s: %v", l.code, err)
		}
		fmt.Printf("seeded %s (%d clicks)\n", l.code, clicks)
	}

	fmt.Println("seed complete")
}

// classify mirrors repositories.parseFields; keep both in sync.
func classify(ua string) (device, browser, os string) {
	lower := strings.ToLower(ua)
	for _, b := range []string{"bot", "crawler", "curl", "wget"} {
		if strings.Contains(lower, b) {
			return "bot", ua, ua
		}
	}

	device = "desktop"
	switch {
	case strings.Contains(lower, "ipad"):
		device = "tablet"
	case strings.Contains(ua, "Mobile"):
		device = "mobile"
	}

	browser = browsers[rnd.Intn(len(browsers))]
	os = "Other"
	switch {
	case strings.Contains(lower, "windows"):
		os = "Windows"
	case strings.Contains(lower, "mac os"):
		os = "Mac OS X"
	case strings.Contains(lower, "iphone"):
		os = "iOS"
	case strings.Contains(lower, "android"):
		os = "Android"
	}
	return device, browser, os
}
