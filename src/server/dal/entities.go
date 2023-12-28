package dal

import (
	"time"
)

type Account struct {
	Id              int
	CreatedAt       time.Time
	ApproveStatus   int    // <=-100: banned; 1: has been manually approved before
	UserUrl         string // https://rss-parrot.net/u/ludic.mataroa.blog
	Handle          string // ludic.mataroa.blog
	Name            string // Ludicity
	Summary         string // "While I'm deeply sympathetic, the author should be discussing their issues with a therapist rather than spreading this on the internet."
	SiteUrl         string // https://ludic.mataroa.blog/
	FeedUrl         string // https://ludic.mataroa.blog/rss/
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
	Desription   string
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
