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
	AddAccountIfNotExist(account *Account, privKey string) (isNew bool, err error)
	DoesAccountExist(user string) (bool, error)
	GetPrivKey(user string) (string, error)
	GetAccount(user string) (*Account, error)
	GetPostCount(user string) (uint, error)
	GetFeedLastUpdated(accountId int) (time.Time, error)
	UpdateAccountFeedTimes(accountId int, lastUpdated, nextCheckDue time.Time) error
	AddFeedPostIfNew(accountId int, post *FeedPost) (isNew bool, err error)
	GetFollowerCount(user string) (uint, error)
	GetFollowers(user string) ([]*MastodonUserInfo, error)
	AddFollower(user string, follower *MastodonUserInfo) error
	RemoveFollower(user, followerUserUrl string) error
}

type Repo struct {
	cfg    *shared.Config
	logger shared.ILogger
	db     *sql.DB
	muId   sync.Mutex
	nextId uint64
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

	// DBG
	//repo.test()

	return &repo
}

func (repo *Repo) test() {
	//_, err := repo.db.Exec(`INSERT INTO accounts
	//	(created_at, user_url, handle, name, summary, profile_image_url, site_url, rss_url, pubkey)
	//	VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	//	time.Now(), "user-url", "handle", "name", "summary", "profile-pic",
	//	"site-url", "rss-url", "pub-key")
	//if err != nil {
	//	repo.logger.Errorf("%v", err)
	//}
	//repo.logger.Info("test")
}

func (repo *Repo) GetNextId() uint64 {
	repo.muId.Lock()
	res := repo.nextId + 1
	repo.nextId = res
	repo.muId.Unlock()
	return res
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

	_, err := repo.db.Exec(`INSERT INTO accounts
    	(created_at, user_url, handle, pubkey, privkey)
		VALUES(?, ?, ?, ?, ?)`,
		repo.cfg.Birb.Published, idb.UserUrl(repo.cfg.Birb.User),
		repo.cfg.Birb.User, repo.cfg.Birb.PubKey, repo.cfg.Birb.PrivKey)

	if err != nil {
		repo.logger.Errorf("Failed to add built-in user '%s': %v", repo.cfg.Birb.User, err)
		panic(err)
	}
}

func (repo *Repo) AddAccountIfNotExist(acct *Account, privKey string) (isNew bool, err error) {
	isNew = true
	_, err = repo.db.Exec(`INSERT INTO accounts
    	(created_at, user_url, handle, name, summary, profile_image_url, site_url, rss_url, pubkey, privkey)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		acct.CreatedAt, acct.UserUrl, acct.Handle, acct.Name, acct.Summary, acct.ProfileImageUrl,
		acct.SiteUrl, acct.FeedUrl, acct.PubKey, privKey)
	if err == nil {
		return
	}
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		if mysqlErr.Number == 1062 { // Duplicate key: account with this handle already exists
			isNew = false
			_, err = repo.GetAccount(acct.Handle)
			return
		}
	}
	return
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
		`SELECT id, created_at, user_url, handle, name, summary, profile_image_url, rss_url,
         		feed_last_updated, next_check_due, pubkey
		FROM accounts WHERE handle=?`, user)
	var err error
	var res Account
	err = row.Scan(&res.Id, &res.CreatedAt, &res.UserUrl, &res.Handle, &res.Name, &res.Summary,
		&res.ProfileImageUrl, &res.FeedUrl, &res.FeedLastUpdated, &res.NextCheckDue, &res.PubKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &res, nil
}

func (repo *Repo) GetPrivKey(user string) (string, error) {
	row := repo.db.QueryRow(`SELECT privkey FROM accounts WHERE handle=?`, user)
	var err error
	var res string
	err = row.Scan(&res)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		} else {
			return "", err
		}
	}
	return res, nil
}

func (repo *Repo) SetPrivKey(user, privKey string) error {
	_, err := repo.db.Exec("UPDATE accounts SET privkey=? WHERE handle=?", privKey, user)
	if err != nil {
		return err
	}
	return nil
}

func (repo *Repo) GetPostCount(user string) (uint, error) {
	return 734, nil
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

func (repo *Repo) GetFeedLastUpdated(accountId int) (res time.Time, err error) {
	res = time.Time{}
	err = nil
	row := repo.db.QueryRow("SELECT feed_last_updated FROM accounts WHERE id=?", accountId)
	if err = row.Scan(&res); err != nil {
		return
	}
	return
}

func (repo *Repo) UpdateAccountFeedTimes(accountId int, lastUpdated, nextCheckDue time.Time) error {
	_, err := repo.db.Exec(`UPDATE accounts SET feed_last_updated=?, next_check_due=?
        WHERE id=?`, lastUpdated, nextCheckDue, accountId)
	return err
}

func (repo *Repo) AddFeedPostIfNew(accountId int, post *FeedPost) (isNew bool, err error) {

	err = nil

	_, err = repo.db.Exec(`INSERT INTO feed_posts
    	(account_id, post_guid_hash, post_time, link, title, description)
		VALUES (?, ?, ?, ?, ?, ?)`,
		accountId, post.PostGuidHash, post.PostTime, post.Link, post.Title, post.Desription)

	if err == nil {
		isNew = true
		return
	}

	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		if mysqlErr.Number == 1062 { // Duplicate key: account with this handle already exists
			isNew = false
			err = nil
			return
		}
	}

	return
}
