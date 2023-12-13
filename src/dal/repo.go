package dal

import (
	"rss_parrot/shared"
	"sync"
	"time"
)

type IRepo interface {
	GetNextId() uint64
	GetPostCount() uint
	GetPosts() []*Post
	AddPost(post *Post)
	GetFollowers() []*Follower
	AddFollower(follower *Follower)
	RemoveFollower(user string)
}

type Repo struct {
	cfg       *shared.Config
	posts     []*Post
	followers []*Follower
	muId      sync.Mutex
	nextId    uint64
}

func NewRepo(cfg *shared.Config) IRepo {
	repo := Repo{
		cfg:       cfg,
		posts:     []*Post{},
		followers: []*Follower{},
		nextId:    uint64(time.Now().UnixMilli()),
	}
	repo.addInitialData()
	return &repo
}

func (repo *Repo) addInitialData() {
	t, _ := time.Parse(time.RFC3339, "2023-12-11T21:15:01Z")
	repo.posts = append(repo.posts, &Post{"First post", t})
	t, _ = time.Parse(time.RFC3339, "2023-12-11T21:19:05Z")
	repo.posts = append(repo.posts, &Post{"Second post", t})
	t, _ = time.Parse(time.RFC3339, "2023-12-11T21:21:37Z")
	repo.posts = append(repo.posts, &Post{"And it's stopped raining!", t})
	repo.followers = append(repo.followers, &Follower{
		"https://genart.social/users/twilliability", "twilliability",
		"genart.social", "https://genart.social/inbox"})
}

func (repo *Repo) GetNextId() uint64 {
	repo.muId.Lock()
	res := repo.nextId + 1
	repo.nextId = res
	repo.muId.Unlock()
	return res
}

func (repo *Repo) GetPostCount() uint {
	return uint(len(repo.posts))
}

func (repo *Repo) GetPosts() []*Post {
	return repo.posts
}

func (repo *Repo) AddPost(post *Post) {
	repo.posts = append(repo.posts, post)
}

func (repo *Repo) GetFollowers() []*Follower {
	return repo.followers
}

func (repo *Repo) AddFollower(follower *Follower) {
	for _, f := range repo.followers {
		if f.User == follower.User {
			return
		}
	}
	repo.followers = append(repo.followers, follower)
}

func (repo *Repo) RemoveFollower(user string) {
	new := make([]*Follower, 0, len(repo.followers))
	for _, f := range repo.followers {
		if f.User != user {
			new = append(new, f)
		}
	}
	repo.followers = new
}
