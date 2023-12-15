package dal

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"rss_parrot/shared"
	"sync"
	"time"
)

const schemaVer = 1

//go:embed scripts/*
var scripts embed.FS

type IRepo interface {
	InitUpdateDb()
	GetNextId() uint64
	GetPostCount() uint
	GetPosts() []*Post
	AddPost(post *Post)
	GetFollowerCount() uint
	GetFollowers() []*Follower
	AddFollower(follower *Follower)
	RemoveFollower(user string)
}

type Repo struct {
	cfg       *shared.Config
	logger    shared.ILogger
	db        *sql.DB
	posts     []*Post
	followers []*Follower
	muId      sync.Mutex
	nextId    uint64
}

func NewRepo(cfg *shared.Config, logger shared.ILogger) IRepo {

	// Connect to DB
	sqlCfg := mysql.Config{
		User:            cfg.Secrets.DbUser,
		Passwd:          cfg.Secrets.DbPass,
		Net:             cfg.Db.Net,
		Addr:            cfg.Db.Addr,
		DBName:          cfg.Db.DbName,
		MultiStatements: true,
	}
	var err error
	var db *sql.DB
	if db, err = sql.Open("mysql", sqlCfg.FormatDSN()); err != nil {
		logger.Errorf("Failed to connect to DB: %v", err)
		panic(err)
	}
	if err = db.Ping(); err != nil {
		logger.Errorf("Failed to ping DB: %v", err)
		panic(err)
	}

	repo := Repo{
		cfg:       cfg,
		logger:    logger,
		db:        db,
		posts:     []*Post{},
		followers: []*Follower{},
		nextId:    uint64(time.Now().UnixMilli()),
	}
	repo.addInitialData()
	return &repo
}

func (repo *Repo) InitUpdateDb() {
	dbVer := 0
	sysParamsExists := false
	var err error
	var rows *sql.Rows
	if rows, err = repo.db.Query("SHOW TABLES LIKE 'sys_params'"); err != nil {
		repo.logger.Errorf("Failed to check if 'sys_params' table exists: %v", err)
		panic(err)
	}
	for rows.Next() {
		sysParamsExists = true
	}
	_ = rows.Close()
	if !sysParamsExists {
		repo.logger.Printf("Database appears to be empty; current schema version is %d", schemaVer)
	} else {
		row := repo.db.QueryRow("SELECT val FROM sys_params WHERE name='schema_ver'")
		if err = row.Scan(&dbVer); err != nil {
			repo.logger.Errorf("Failed to query schema version: %v", err)
			panic(err)
		}
		repo.logger.Printf("Database is at version %d; current schema version is %d", dbVer, schemaVer)
	}
	for i := dbVer; i < schemaVer; i += 1 {
		nextVer := i + 1
		fn := fmt.Sprintf("scripts/create-%02d.sql", nextVer)
		repo.logger.Printf("Running %s", fn)
		var sqlBytes []byte
		if sqlBytes, err = scripts.ReadFile(fn); err != nil {
			repo.logger.Errorf("Failed to read init script %s: %v", fn, err)
			panic(err)
		}
		sqlStr := string(sqlBytes)
		if _, err = repo.db.Exec(sqlStr); err != nil {
			repo.logger.Errorf("Failed to execute init script %s: %v", fn, err)
			panic(err)
		}
		_, err = repo.db.Exec("UPDATE sys_params SET val=? WHERE name='schema_ver'", nextVer)
		if err != nil {
			repo.logger.Errorf("Failed to update schema_ver to %d: %v", i, err)
			panic(err)
		}
	}
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

func (repo *Repo) GetFollowerCount() uint {
	return uint(len(repo.followers))
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
