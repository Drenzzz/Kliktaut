package services

import (
	"kliktaut-api/internal/repositories"

	"gorm.io/gorm"
)

type AnalyticsService struct {
	analytics *repositories.AnalyticsRepository
	links     *repositories.LinkRepository
}

func NewAnalyticsService(a *repositories.AnalyticsRepository, l *repositories.LinkRepository) *AnalyticsService {
	return &AnalyticsService{analytics: a, links: l}
}

// LinkAnalytics is the response shape for GET /api/v1/links/:id/analytics.
type LinkAnalytics struct {
	LinkID          string                     `json:"link_id"`
	TotalClicks     int64                      `json:"total_clicks"`
	UniqueVisitors  int64                      `json:"unique_visitors"`
	ClicksByDevice  map[string]int64           `json:"clicks_by_device"`
	ClicksByBrowser map[string]int64           `json:"clicks_by_browser"`
	TopReferrers    []repositories.ReferrerRow `json:"top_referrers"`
	ClicksLast7Days []repositories.DayRow      `json:"clicks_last_7_days"`
}

func (s *AnalyticsService) ForLink(linkID string) (*LinkAnalytics, error) {
	sum, err := s.analytics.Summary(linkID)
	if err != nil {
		return nil, err
	}
	return &LinkAnalytics{
		LinkID:          linkID,
		TotalClicks:     sum.TotalClicks,
		UniqueVisitors:  sum.UniqueVisitors,
		ClicksByDevice:  sum.ByDevice,
		ClicksByBrowser: sum.ByBrowser,
		TopReferrers:    sum.TopReferrers,
		ClicksLast7Days: sum.Last7Days,
	}, nil
}

func (s *AnalyticsService) ClicksList(linkID string, page, perPage int) ([]map[string]interface{}, int64, error) {
	return s.analytics.ClicksList(linkID, page, perPage)
}

func (s *AnalyticsService) Overview() (map[string]interface{}, error) {
	return s.analytics.Overview()
}

// Links exposes the link repository for existence checks in handlers.
func (s *AnalyticsService) Links() *repositories.LinkRepository {
	return s.links
}

// ForLinkExist checks link existence, translating not-found to ErrLinkNotFound.
func (s *AnalyticsService) ForLinkExist(id string) (bool, error) {
	_, err := s.links.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, ErrLinkNotFound
		}
		return false, err
	}
	return true, nil
}
