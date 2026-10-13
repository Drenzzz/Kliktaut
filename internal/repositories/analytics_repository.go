package repositories

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type AnalyticsRepository struct {
	DB *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) *AnalyticsRepository {
	return &AnalyticsRepository{DB: db}
}

type CountRow struct {
	Key   string  `json:"key"`
	Count int64   `json:"count"`
	IP    *string `json:"-"`
}

// Summary returns all metrics for one link. Bot exclusion rules (§6.4):
// total_clicks & unique_visitors exclude device_type='bot';
// clicks_by_device INCLUDES bot (RISK-011, documented behavior).
type Summary struct {
	TotalClicks    int64
	UniqueVisitors int64
	ByDevice       map[string]int64
	ByBrowser      map[string]int64
	TopReferrers   []ReferrerRow
	Last7Days      []DayRow
}

type ReferrerRow struct {
	Referrer string `json:"referrer"`
	Count    int64  `json:"count"`
}

type DayRow struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

func (r *AnalyticsRepository) Summary(linkID string) (*Summary, error) {
	s := &Summary{
		ByDevice:     map[string]int64{},
		ByBrowser:    map[string]int64{},
		TopReferrers: []ReferrerRow{},
		Last7Days:    []DayRow{},
	}

	if err := r.DB.Raw(
		`SELECT COUNT(*) FROM clicks WHERE link_id = ? AND (device_type IS DISTINCT FROM 'bot')`,
		linkID,
	).Scan(&s.TotalClicks).Error; err != nil {
		return nil, err
	}

	if err := r.DB.Raw(
		`SELECT COUNT(DISTINCT ip_address) FROM clicks WHERE link_id = ? AND (device_type IS DISTINCT FROM 'bot')`,
		linkID,
	).Scan(&s.UniqueVisitors).Error; err != nil {
		return nil, err
	}

	byDevice, err := r.groupCount(`SELECT device_type AS key, COUNT(*) AS count FROM clicks WHERE link_id = ? GROUP BY device_type`, linkID)
	if err != nil {
		return nil, err
	}
	s.ByDevice = byDevice

	byBrowser, err := r.groupCount(`SELECT browser AS key, COUNT(*) AS count FROM clicks WHERE link_id = ? GROUP BY browser ORDER BY count DESC LIMIT 10`, linkID)
	if err != nil {
		return nil, err
	}
	s.ByBrowser = byBrowser

	rows := []struct {
		Referrer *string
		Count    int64
	}{}
	if err := r.DB.Raw(
		`SELECT referrer, COUNT(*) AS count FROM clicks WHERE link_id = ? GROUP BY referrer ORDER BY count DESC LIMIT 10`,
		linkID,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		ref := "direct"
		if row.Referrer != nil && *row.Referrer != "" {
			ref = *row.Referrer
		}
		s.TopReferrers = append(s.TopReferrers, ReferrerRow{Referrer: ref, Count: row.Count})
	}

	days := []struct {
		Day   time.Time
		Count int64
	}{}
	if err := r.DB.Raw(
		`SELECT date_trunc('day', clicked_at) AS day, COUNT(*) AS count
		 FROM clicks
		 WHERE link_id = ? AND clicked_at >= date_trunc('day', now() AT TIME ZONE current_setting('TimeZone')) - interval '6 days'
		 GROUP BY day ORDER BY day`,
		linkID,
	).Scan(&days).Error; err != nil {
		return nil, err
	}
	// fill all 7 days so the client gets a continuous series
	dayMap := map[string]int64{}
	for _, d := range days {
		dayMap[d.Day.Format("2006-01-02")] = d.Count
	}
	today := time.Now().Truncate(24 * time.Hour)
	for i := 6; i >= 0; i-- {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")
		s.Last7Days = append(s.Last7Days, DayRow{Date: day, Count: dayMap[day]})
	}

	return s, nil
}

func (r *AnalyticsRepository) groupCount(query string, args ...interface{}) (map[string]int64, error) {
	rows := []CountRow{}
	if err := r.DB.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("group count: %w", err)
	}
	out := map[string]int64{}
	for _, row := range rows {
		key := row.Key
		if key == "" {
			key = "unknown"
		}
		out[key] = row.Count
	}
	return out, nil
}

// ClicksList returns paginated raw clicks for one link.
func (r *AnalyticsRepository) ClicksList(linkID string, page, perPage int) ([]map[string]interface{}, int64, error) {
	var total int64
	if err := r.DB.Raw(`SELECT COUNT(*) FROM clicks WHERE link_id = ?`, linkID).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	var clicks []map[string]interface{}
	err := r.DB.Table("clicks").
		Select(`id, ip_address, user_agent, referrer, country, device_type, browser, os, clicked_at`).
		Where("link_id = ?", linkID).
		Order("clicked_at DESC").
		Limit(perPage).
		Offset((page - 1) * perPage).
		Find(&clicks).Error
	if err != nil {
		return nil, 0, err
	}

	return clicks, total, nil
}

// Overview returns /api/v1/stats metrics (§6.4). Day boundaries use the
// DB session timezone (decision OQ-007).
func (r *AnalyticsRepository) Overview() (map[string]interface{}, error) {
	out := map[string]interface{}{}

	if err := r.DB.Raw(
		`SELECT
			COUNT(*) AS total_links,
			COUNT(*) FILTER (WHERE is_active AND (expires_at IS NULL OR expires_at >= now())) AS active_links,
			COUNT(*) FILTER (WHERE is_active AND expires_at < now()) AS expired_links,
			COUNT(*) FILTER (WHERE NOT is_active) AS deleted_links
		 FROM links`,
	).Scan(&out).Error; err != nil {
		return nil, err
	}

	if err := r.DB.Raw(
		`SELECT
			COUNT(*) AS total_clicks,
			COUNT(*) FILTER (WHERE clicked_at >= date_trunc('day', now() AT TIME ZONE current_setting('TimeZone'))) AS clicks_today
		 FROM clicks`,
	).Scan(&out).Error; err != nil {
		return nil, err
	}

	days := []struct {
		Day   time.Time
		Count int64
	}{}
	if err := r.DB.Raw(
		`SELECT date_trunc('day', clicked_at) AS day, COUNT(*) AS count
		 FROM clicks
		 WHERE clicked_at >= date_trunc('day', now() AT TIME ZONE current_setting('TimeZone')) - interval '6 days'
		 GROUP BY day ORDER BY day`,
	).Scan(&days).Error; err != nil {
		return nil, err
	}
	dayMap := map[string]int64{}
	for _, d := range days {
		dayMap[d.Day.Format("2006-01-02")] = d.Count
	}
	series := []DayRow{}
	today := time.Now().Truncate(24 * time.Hour)
	for i := 6; i >= 0; i-- {
		day := today.AddDate(0, 0, -i).Format("2006-01-02")
		series = append(series, DayRow{Date: day, Count: dayMap[day]})
	}
	out["clicks_last_7_days"] = series

	type TopLink struct {
		ShortCode  string
		Title      *string
		ClickCount int64
	}
	top := []TopLink{}
	if err := r.DB.Raw(
		`SELECT short_code, title, click_count FROM links ORDER BY click_count DESC LIMIT 10`,
	).Scan(&top).Error; err != nil {
		return nil, err
	}
	out["top_links"] = top

	return out, nil
}
