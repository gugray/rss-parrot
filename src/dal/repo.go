package dal

import (
	"database/sql"
	"embed"
	"errors"
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
	AddAccount(account *Account) error
	DoesAccountExist(user string) (bool, error)
	GetAccount(user string) (*Account, error)
	GetPostCount(user string) (uint, error)
	GetPosts(user string) ([]*Post, error)
	AddPost(post *Post) error
	GetFollowerCount(user string) (uint, error)
	GetFollowers(user string) ([]*MastodonUserInfo, error)
	AddFollower(user string, follower *MastodonUserInfo) error
	RemoveFollower(user, followerUserUrl string) error
}

type Repo struct {
	cfg       *shared.Config
	logger    shared.ILogger
	db        *sql.DB
	posts     []*Post
	followers []*MastodonUserInfo
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
		ParseTime:       true,
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
		cfg:    cfg,
		logger: logger,
		db:     db,
		nextId: uint64(time.Now().UnixNano()),
	}

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

	if dbVer == 0 {
		repo.mustAddBuiltInUsers()
	}
}

func (repo *Repo) mustAddBuiltInUsers() {
	idb := shared.IdBuilder{Host: repo.cfg.Host}
	err := repo.AddAccount(&Account{
		CreatedAt:       repo.cfg.Birb.Published,
		UserUrl:         idb.UserUrl(repo.cfg.Birb.User),
		Handle:          repo.cfg.Birb.User,
		Name:            repo.cfg.Birb.Name,
		Summary:         repo.cfg.Birb.Summary,
		ProfileImageUrl: repo.cfg.Birb.ProfilePic,
		RssUrl:          "",
		PubKey:          repo.cfg.Birb.PubKey,
		PrivKey:         repo.cfg.Birb.PrivKey,
	})
	if err != nil {
		repo.logger.Errorf("Failed to add built-in users: %v", err)
		panic(err)
	}
}

func (repo *Repo) AddAccount(acct *Account) error {
	_, err := repo.db.Exec(`INSERT INTO accounts
    	(created_at, user_url, handle, name, summary, profile_image_url, site_url, rss_url, pubkey, privkey)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		acct.CreatedAt, acct.UserUrl, acct.Handle, acct.Name, acct.Summary, acct.ProfileImageUrl,
		acct.SiteUrl, acct.RssUrl, acct.PubKey, acct.PrivKey)
	if err != nil {
		return err
	}
	return nil
}

func (repo *Repo) DoesAccountExist(user string) (bool, error) {
	row := repo.db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE handle=?`, user)
	var err error
	var count int
	if err = row.Scan(&count); err != nil {
		return false, err
	}
	return count != 0, nil
}

func (repo *Repo) GetAccount(user string) (*Account, error) {
	row := repo.db.QueryRow(
		`SELECT created_at, user_url, handle, name, summary, profile_image_url, rss_url, pubkey, privkey
		FROM accounts WHERE handle=?`, user)
	var err error
	var res Account
	err = row.Scan(&res.CreatedAt, &res.UserUrl, &res.Handle, &res.Name, &res.Summary, &res.ProfileImageUrl,
		&res.RssUrl, &res.PubKey, &res.PrivKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &res, nil
}

func (repo *Repo) GetNextId() uint64 {
	repo.muId.Lock()
	res := repo.nextId + 1
	repo.nextId = res
	repo.muId.Unlock()
	return res
}

func (repo *Repo) GetPostCount(user string) (uint, error) {
	return 734, nil
}

func (repo *Repo) GetPosts(user string) ([]*Post, error) {
	return []*Post{}, nil
}

func (repo *Repo) AddPost(post *Post) error {
	return nil
}

func (repo *Repo) GetFollowerCount(user string) (uint, error) {
	row := repo.db.QueryRow(`SELECT COUNT(*) FROM followers JOIN accounts
		ON followers.account_id=accounts.id AND accounts.handle=?`, user)
	var err error
	var count int
	if err = row.Scan(&count); err != nil {
		return 0, err
	}
	return uint(count), nil
}

func (repo *Repo) GetFollowers(user string) ([]*MastodonUserInfo, error) {
	rows, err := repo.db.Query(`SELECT followers.user_url, followers.handle, host, shared_inbox
		FROM followers JOIN accounts ON followers.account_id=accounts.id AND accounts.handle=?`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := make([]*MastodonUserInfo, 0)
	for rows.Next() {
		mui := MastodonUserInfo{}
		if err = rows.Scan(&mui.UserUrl, &mui.Handle, &mui.Host, &mui.SharedInbox); err != nil {
			return nil, err
		}
		res = append(res, &mui)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func (repo *Repo) AddFollower(user string, follower *MastodonUserInfo) error {
	row := repo.db.QueryRow(`SELECT id FROM accounts WHERE handle=?`, user)
	var err error
	var accountId int
	if err = row.Scan(&accountId); err != nil {
		return err
	}
	_, err = repo.db.Exec(`INSERT INTO followers VALUES(?, ?, ?, ?, ?)`,
		accountId, follower.UserUrl, follower.Handle, follower.Host, follower.SharedInbox)
	if err != nil {
		return err
	}
	return nil
}

func (repo *Repo) RemoveFollower(user, followerUserUrl string) error {
	row := repo.db.QueryRow(`SELECT id FROM accounts WHERE handle=?`, user)
	var err error
	var accountId int
	if err = row.Scan(&accountId); err != nil {
		return err
	}
	_, err = repo.db.Exec(`DELETE FROM followers WHERE account_id=? AND user_url=?`,
		accountId, followerUserUrl)
	if err != nil {
		return err
	}
	return nil
}
