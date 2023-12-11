package dal

import "time"

type Post struct {
	Content   string
	Published time.Time
}

type Follower struct {
	User   string // https://genart.social/users/twilliability
	Handle string // twilliability
	Host   string // genart.social
}
