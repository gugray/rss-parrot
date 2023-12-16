package dal

import (
	"time"
)

type Account struct {
	CreatedAt       time.Time
	UserUrl         string // https://rss-parrot.net/u/ludic.mataroa.blog
	Handle          string // ludic.mataroa.blog
	Name            string // Ludicity
	Summary         string // "While I'm deeply sympathetic, the author should be discussing their issues with a therapist rather than spreading this on the internet."
	SiteUrl         string // https://ludic.mataroa.blog/
	RssUrl          string // https://ludic.mataroa.blog/rss/
	PubKey          string
	PrivKey         string
	ProfileImageUrl string
}

type Mention struct {
	StatusIdUrl string
	UserInfo    *MastodonUserInfo
}

type Post struct {
	StatusId  string
	Content   string
	Published time.Time
}

type MastodonUserInfo struct {
	UserUrl     string // https://genart.social/users/twilliability
	Handle      string // twilliability
	Host        string // genart.social
	SharedInbox string // https://genart.social/inbox
}
