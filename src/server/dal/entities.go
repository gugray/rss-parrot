package dal

import (
	"time"
)

type Account struct {
	Id              int
	CreatedAt       time.Time
	UserUrl         string // https://rss-parrot.net/u/taiwantrailsandtales.com
	Handle          string // taiwantrailsandtales.com
	FeedName        string // taiwan trails and tales | a guide to get you out of the city and into the hills
	FeedSummary     string // Taiwan Trails and Tales is a one-stop shop for everything Taiwan hiking related. Here you can find information about hundreds of hiking trails in Taiwan, as well as all the details you need to know about how and when to visit.
	SiteUrl         string // https://taiwantrailsandtales.com
	FeedUrl         string // https://taiwantrailsandtales.com/feed
	FeedLastUpdated time.Time
	NextCheckDue    time.Time
	PubKey          string
	ProfileImageUrl string
	HeaderImageUrl  string
}

type Mention struct {
	StatusIdUrl string
	UserInfo    *FollowerInfo
}

type FeedPost struct {
	PostGuidHash int64
	PostTime     time.Time
	Link         string
	Title        string
	Description  string
}

type Toot struct {
	PostGuidHash int64
	TootedAt     time.Time
	StatusId     string
	Content      string
}

type TootQueueItem struct {
	Id          int
	SendingUser string
	ToInbox     string
	TootedAt    time.Time
	StatusId    string
	Content     string
}

type FollowerInfo struct {
	RequestId     string // ID of the follow request activity; needed for approve reply
	ApproveStatus int    // 0: unapproved, 1: approved, negative: banned
	UserUrl       string // https://genart.social/users/twilliability
	Handle        string // twilliability
	Host          string // genart.social
	UserInbox     string // https://genart.social/users/twilliability/inbox
	SharedInbox   string // https://genart.social/inbox
}
