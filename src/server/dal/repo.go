package dal

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
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
	GetAccountsPage(offset, limit int) ([]*Account, int, error)
	GetTootCount(user string) (uint, error)
	AddToot(accountId int, toot *Toot) error
	GetFeedLastUpdated(accountId int) (time.Time, error)
	UpdateAccountFeedTimes(accountId int, lastUpdated, nextCheckDue time.Time) error
	AddFeedPostIfNew(accountId int, post *FeedPost) (isNew bool, err error)
	GetAccountToCheck(checkDue time.Time) (*Account, error)
	GetApprovedFollowerCount(user string) (uint, error)
	GetFollowersByUser(user string, onlyApproved bool) ([]*FollowerInfo, error)
	GetFollowersById(accountId int, onlyApproved bool) ([]*FollowerInfo, error)
	SetFollowerApproveStatus(user, followerUserUrl string, status int) error
	AddFollower(user string, follower *FollowerInfo) error
	RemoveFollower(user, followerUserUrl string) error
	AddTootQueueItem(tqi *TootQueueItem) error
	GetTootQueueItems(aboveId, maxCount int) ([]*TootQueueItem, error)
	DeleteTootQueueItem(id int) error
	MarkActivityHandled(id string, when time.Time) (alreadyHandled bool, err error)
}

type Repo struct {
	cfg    *shared.Config
	logger shared.ILogger
	db     *sql.DB
	muDb   sync.RWMutex
	muId   sync.Mutex
	nextId uint64
}

func NewRepo(cfg *shared.Config, logger shared.ILogger) IRepo {

	var err error
	var db *sql.DB

	// https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
	// https://www.reddit.com/r/golang/comments/16xswxd/concurrency_when_writing_data_into_sqlite/
	// https://github.com/mattn/go-sqlite3/issues/1022#issuecomment-1067353980
	// _synchronous=1 is "normal"
	cstr := "file:%s?cache=shared&mode=rwc&_journal_mode=WAL&_synchronous=1&_busy_timeout=5000"
	db, err = sql.Open("sqlite3", fmt.Sprintf(cstr, cfg.DbFile))
	if err != nil {
		logger.Errorf("Failed to open/create DB file: %s: %v", cfg.DbFile, err)
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

	rows, err = repo.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='sys_params'")
	if err != nil {
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

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	isNew = true
	_, err = repo.db.Exec(`INSERT INTO accounts
    	(created_at, approve_status, user_url, handle, name, summary, profile_image_url, site_url, feed_url, pubkey, privkey)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		acct.CreatedAt, acct.ApproveStatus, acct.UserUrl, acct.Handle, acct.Name, acct.Summary, acct.ProfileImageUrl,
		acct.SiteUrl, acct.FeedUrl, acct.PubKey, privKey)
	if err == nil {
		return
	}
	// MySQL: mysql.MySQLError; mysqlErr.Number == 1062
	if sqliteErr, ok := err.(sqlite3.Error); ok {
		// Duplicate key: account with this handle already exists
		if sqliteErr.Code == 19 && sqliteErr.ExtendedCode == 2067 {
			isNew = false
			_, err = repo.getAccount(acct.Handle)
			return
		}
	}
	return
}

func (repo *Repo) DoesAccountExist(user string) (bool, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	row := repo.db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE handle=?`, user)
	var err error
	var count int
	if err = row.Scan(&count); err != nil {
		return false, err
	}
	return count != 0, nil
}

func (repo *Repo) GetAccount(user string) (*Account, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	return repo.getAccount(user)
}

func (repo *Repo) getAccount(user string) (*Account, error) {

	row := repo.db.QueryRow(
		`SELECT id, created_at, approve_status, user_url, handle, name, summary, profile_image_url, site_url, feed_url,
         		feed_last_updated, next_check_due, pubkey
		FROM accounts WHERE handle=?`, user)
	var err error
	var res Account
	err = row.Scan(&res.Id, &res.CreatedAt, &res.ApproveStatus, &res.UserUrl, &res.Handle, &res.Name, &res.Summary,
		&res.ProfileImageUrl, &res.SiteUrl, &res.FeedUrl, &res.FeedLastUpdated, &res.NextCheckDue, &res.PubKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &res, nil
}

func (repo *Repo) GetAccountsPage(offset, limit int) ([]*Account, int, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	var res []*Account
	var total int
	var err error

	row := repo.db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE approve_status>-100 AND feed_url<>''`)
	if err = row.Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, created_at, approve_status, user_url, handle, name, summary, profile_image_url, site_url, feed_url,
        feed_last_updated, next_check_due, pubkey
		FROM accounts WHERE approve_status>-100 AND feed_url<>'' ORDER BY ID DESC LIMIT ? OFFSET ?`
	rows, err := repo.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		a := Account{}
		err = rows.Scan(&a.Id, &a.CreatedAt, &a.ApproveStatus, &a.UserUrl, &a.Handle, &a.Name, &a.Summary,
			&a.ProfileImageUrl, &a.SiteUrl, &a.FeedUrl, &a.FeedLastUpdated, &a.NextCheckDue, &a.PubKey)
		if err = rows.Err(); err != nil {
			return nil, 0, err
		}
		res = append(res, &a)
	}
	return res, total, nil
}

func (repo *Repo) GetPrivKey(user string) (string, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

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

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	_, err := repo.db.Exec("UPDATE accounts SET privkey=? WHERE handle=?", privKey, user)
	if err != nil {
		return err
	}
	return nil
}

func (repo *Repo) GetTootCount(user string) (uint, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	row := repo.db.QueryRow(`SELECT COUNT(*) FROM toots JOIN accounts
		ON toots.account_id=accounts.id AND accounts.handle=?`, user)
	var err error
	var count int
	if err = row.Scan(&count); err != nil {
		return 0, err
	}
	return uint(count), nil
}

func (repo *Repo) AddToot(accountId int, toot *Toot) error {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	_, err := repo.db.Exec(`INSERT INTO toots (account_id, post_guid_hash, tooted_at, status_id, content)
		VALUES(?, ?, ?, ?, ?)`,
		accountId, toot.PostGuidHash, toot.TootedAt, toot.StatusId, toot.Content)
	if err != nil {
		return err
	}
	return nil
}

func (repo *Repo) GetApprovedFollowerCount(user string) (uint, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	row := repo.db.QueryRow(`SELECT COUNT(*) FROM followers JOIN accounts
		ON followers.account_id=accounts.id AND accounts.handle=?
		WHERE followers.approve_status=1`, user)
	var err error
	var count int
	if err = row.Scan(&count); err != nil {
		return 0, err
	}
	return uint(count), nil
}

func (repo *Repo) SetFollowerApproveStatus(user, followerUserUrl string, status int) error {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	acct, err := repo.getAccount(user)
	if err != nil {
		return err
	}
	_, err = repo.db.Exec(`UPDATE followers SET approve_status=? WHERE account_id=? AND user_url=?`,
		status, acct.Id, followerUserUrl)
	if err != nil {
		return err
	}
	return nil
}

func (repo *Repo) GetFollowersByUser(user string, onlyApproved bool) ([]*FollowerInfo, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	query := `SELECT followers.request_id, followers.user_url, followers.handle, host, user_inbox, shared_inbox
		FROM followers JOIN accounts ON followers.account_id=accounts.id AND accounts.handle=?`
	if onlyApproved {
		query += ` WHERE followers.approve_status=1`
	}
	rows, err := repo.db.Query(query, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readGetFollowers(rows)
}

func (repo *Repo) GetFollowersById(accountId int, onlyApproved bool) ([]*FollowerInfo, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	query := `SELECT request_id, user_url, handle, host, user_inbox, shared_inbox FROM followers WHERE account_id=?`
	if onlyApproved {
		query += ` AND followers.approve_status=1`
	}
	rows, err := repo.db.Query(query, accountId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return readGetFollowers(rows)
}

func readGetFollowers(rows *sql.Rows) ([]*FollowerInfo, error) {
	var err error
	res := make([]*FollowerInfo, 0)
	for rows.Next() {
		mui := FollowerInfo{}
		err = rows.Scan(&mui.RequestId, &mui.UserUrl, &mui.Handle, &mui.Host, &mui.UserInbox, &mui.SharedInbox)
		if err != nil {
			return nil, err
		}
		res = append(res, &mui)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func (repo *Repo) AddFollower(user string, flwr *FollowerInfo) error {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	row := repo.db.QueryRow(`SELECT id FROM accounts WHERE handle=?`, user)
	var err error
	var accountId int
	if err = row.Scan(&accountId); err != nil {
		return err
	}
	_, err = repo.db.Exec(`INSERT INTO followers VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO UPDATE SET request_id=excluded.request_id, approve_status=excluded.approve_status`,
		accountId, flwr.RequestId, flwr.ApproveStatus, flwr.UserUrl, flwr.Handle, flwr.Host,
		flwr.UserInbox, flwr.SharedInbox)
	if err != nil {
		return err
	}
	return nil
}

func (repo *Repo) RemoveFollower(user, followerUserUrl string) error {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

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

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	res = time.Time{}
	err = nil
	row := repo.db.QueryRow("SELECT feed_last_updated FROM accounts WHERE id=?", accountId)
	if err = row.Scan(&res); err != nil {
		return
	}
	return
}

func (repo *Repo) UpdateAccountFeedTimes(accountId int, lastUpdated, nextCheckDue time.Time) error {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	_, err := repo.db.Exec(`UPDATE accounts SET feed_last_updated=?, next_check_due=?
        WHERE id=?`, lastUpdated, nextCheckDue, accountId)
	return err
}

func (repo *Repo) GetAccountToCheck(checkDue time.Time) (*Account, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	rows, err := repo.db.Query(`SELECT id, created_at, approve_status, user_url, handle, name, summary,
    	profile_image_url, site_url, feed_url, feed_last_updated, next_check_due, pubkey
		FROM accounts WHERE next_check_due<? LIMIT 1`, checkDue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var acct *Account = nil
	for rows.Next() {
		res := Account{}
		err = rows.Scan(&res.Id, &res.CreatedAt, &res.ApproveStatus, &res.UserUrl, &res.Handle, &res.Name, &res.Summary,
			&res.ProfileImageUrl, &res.SiteUrl, &res.FeedUrl, &res.FeedLastUpdated, &res.NextCheckDue, &res.PubKey)
		if err = rows.Err(); err != nil {
			return nil, err
		}
		acct = &res
	}
	return acct, nil

}

func (repo *Repo) AddFeedPostIfNew(accountId int, post *FeedPost) (isNew bool, err error) {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	err = nil

	_, err = repo.db.Exec(`INSERT INTO feed_posts
    	(account_id, post_guid_hash, post_time, link, title, description)
		VALUES (?, ?, ?, ?, ?, ?)`,
		accountId, post.PostGuidHash, post.PostTime, post.Link, post.Title, post.Desription)

	if err == nil {
		isNew = true
		return
	}

	// Duplicate key: feed post for this account+guid_hash already exists
	if sqliteErr, ok := err.(sqlite3.Error); ok {
		// Duplicate key: record with this unique key already exists
		if sqliteErr.Code == 19 && sqliteErr.ExtendedCode == 2067 {
			isNew = false
			err = nil
			return
		}
	}

	return
}

func (repo *Repo) AddTootQueueItem(tqi *TootQueueItem) error {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	_, err := repo.db.Exec(`INSERT INTO toot_queue (sending_user, to_inbox, tooted_at, status_id, content)
		VALUES(?, ?, ?, ?, ?)`,
		tqi.SendingUser, tqi.ToInbox, tqi.TootedAt, tqi.StatusId, tqi.Content)
	return err
}

func (repo *Repo) GetTootQueueItems(aboveId, maxCount int) ([]*TootQueueItem, error) {

	repo.muDb.RLock()
	defer repo.muDb.RUnlock()

	rows, err := repo.db.Query(`SELECT id, sending_user, to_inbox, tooted_at, status_id, content
		FROM toot_queue WHERE id>? ORDER BY id ASC LIMIT ?`, aboveId, maxCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := make([]*TootQueueItem, 0, maxCount)
	for rows.Next() {
		tqi := TootQueueItem{}
		err = rows.Scan(&tqi.Id, &tqi.SendingUser, &tqi.ToInbox, &tqi.TootedAt, &tqi.StatusId, &tqi.Content)
		if err != nil {
			return nil, err
		}
		res = append(res, &tqi)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func (repo *Repo) DeleteTootQueueItem(id int) error {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	_, err := repo.db.Exec(`DELETE FROM toot_queue WHERE id=?`, id)
	return err
}

func (repo *Repo) MarkActivityHandled(id string, when time.Time) (alreadyHandled bool, err error) {

	repo.muDb.Lock()
	defer repo.muDb.Unlock()

	alreadyHandled = false
	err = nil

	_, err = repo.db.Exec(`INSERT INTO handled_activities VALUES (?, ?)`, id, when)

	if err == nil {
		return
	}

	// Duplicate key: activity was handled before
	if sqliteErr, ok := err.(sqlite3.Error); ok {
		// Duplicate key: account with this handle already exists
		if sqliteErr.Code == 19 && sqliteErr.ExtendedCode == 2067 {
			alreadyHandled = true
			err = nil
			return
		}
	}

	return

}
