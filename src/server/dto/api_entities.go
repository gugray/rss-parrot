package dto

import "time"

type Feed struct {
	CreatedAt       time.Time `json:"created_at"`
	ApproveStatus   int       `json:"approve_status"`
	UserUrl         string    `json:"user_url"`
	Handle          string    `json:"handle"`
	Name            string    `json:"name"`
	Summary         string    `json:"summary"`
	ProfileImageUrl string    `json:"profile_image_url"`
	SiteUrl         string    `json:"site_url"`
	FeedUrl         string    `json:"feed_url"`
	FeedLastUpdated time.Time `json:"feed_last_updated"`
	NextCheckDue    time.Time `json:"next_check_due"`
}
