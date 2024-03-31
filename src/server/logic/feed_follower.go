package logic

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/spaolacci/murmur3"
	"html"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"rss_parrot/dal"
	"rss_parrot/shared"
	"rss_parrot/texts"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../test/mocks/mock_feed_follower.go -package mocks rss_parrot/logic IFeedFollower

const feedCheckLoopIdleWakeSec = 60
const postPurgeBatchSize = 10

type FeedStatus int32

const (
	FsNew             = 0
	FsAlreadyFollowed = 1
	FsError           = -1
	FsMastodon        = -2
	FsBanned          = -3
	FsOptOut          = -4
)

const (
	feedOrSiteTimeoutSec = 10
)

type IFeedFollower interface {
	GetAccountForFeed(urlStr string) (acct *dal.Account, status FeedStatus, err error)
	PurgeOldPosts(acct *dal.Account, minCount, minAgeDays int) error
}

type SiteInfo struct {
	Url          string
	ParrotHandle string
	FeedUrl      string
	LastUpdated  time.Time
	Title        string
	Description  string
}

type feedFollower struct {
	cfg          *shared.Config
	logger       shared.ILogger
	userAgent    shared.IUserAgent
	repo         dal.IRepo
	blockedFeeds IBlockedFeeds
	messenger    IMessenger
	txt          texts.ITexts
	keyStore     IKeyStore
	metrics      IMetrics
	muDeleting   sync.Mutex
	isDeleting   bool
}

func NewFeedFollower(
	cfg *shared.Config,
	logger shared.ILogger,
	userAgent shared.IUserAgent,
	repo dal.IRepo,
	blockedFeeds IBlockedFeeds,
	messenger IMessenger,
	txt texts.ITexts,
	keyStore IKeyStore,
	metrics IMetrics,
) IFeedFollower {

	ff := feedFollower{
		cfg:          cfg,
		logger:       logger,
		userAgent:    userAgent,
		repo:         repo,
		blockedFeeds: blockedFeeds,
		messenger:    messenger,
		txt:          txt,
		keyStore:     keyStore,
		metrics:      metrics,
		isDeleting:   false,
	}

	ff.updateDBSizeMetric()
	go ff.feedCheckLoop()

	return &ff
}

func (ff *feedFollower) getFeedUrl(siteUrl *url.URL, doc *goquery.Document) string {

	var feedUrlStr string
	isFeedRss := false
	doc.Find("link[rel='alternate']").Each(func(_ int, s *goquery.Selection) {
		var aType, aHref string
		var ok bool
		if aType, ok = s.Attr("type"); !ok {
			return
		}
		if aHref, ok = s.Attr("href"); !ok {
			return
		}
		if aType == "application/atom+xml" && !isFeedRss && feedUrlStr == "" {
			feedUrlStr = aHref
		}
		if aType == "application/rss+xml" && (feedUrlStr == "" || !isFeedRss) {
			feedUrlStr = aHref
			isFeedRss = true
		}
	})

	// Make it absolute
	feedUrl, err := url.Parse(feedUrlStr)
	if err != nil {
		return ""
	}
	if !feedUrl.IsAbs() {
		feedUrl = siteUrl.ResolveReference(feedUrl)
	}

	// Remove (most) query parameters
	ff.trimQueryParams(feedUrl)

	// It's a keeper
	res := feedUrl.String()
	res = strings.TrimRight(res, "/")
	return res
}

func (ff *feedFollower) trimQueryParams(feedUrl *url.URL) {
	// The few exceptions where we keep the query param
	// #33: Youtube feeds look like this: https://www.youtube.com/feeds/videos.xml?channel_id=UCfZz8F37oSJ2rtcEJHM2kCg
	if strings.Contains(feedUrl.Host, "youtube.com") && strings.Contains(feedUrl.RawQuery, "channel_id") {
		return
	}
	// #37: Archive.org: https://archive.org/services/collection-rss.php?collection=misczinespunk
	if strings.Contains(feedUrl.Host, "archive.org") {
		return
	}
	// All otheres: remove query
	feedUrl.RawQuery = ""
}

func (ff *feedFollower) trimQueryParamsStr(urlStr string) (string, error) {
	if parsedUrl, err := url.Parse(urlStr); err != nil {
		return "", err
	} else {
		ff.trimQueryParams(parsedUrl)
		return parsedUrl.String(), nil
	}

}

func (ff *feedFollower) getMetas(doc *goquery.Document, si *SiteInfo) {
	s := doc.Find("title").First()
	if s.Length() != 0 {
		si.Title = s.Text()
	}
	s = doc.Find("meta[name='description']").First()
	if s.Length() != 0 {
		si.Description = s.AttrOr("content", "")
	}
}

func getLastUpdated(feed *gofeed.Feed) time.Time {
	var t time.Time
	if feed.PublishedParsed != nil {
		t = *feed.PublishedParsed
	}
	if feed.UpdatedParsed != nil && feed.UpdatedParsed.After(t) {
		t = *feed.UpdatedParsed
	}
	for _, itm := range feed.Items {
		if itm.PublishedParsed != nil && itm.PublishedParsed.After(t) {
			t = *itm.PublishedParsed
		}
		if itm.UpdatedParsed != nil && itm.UpdatedParsed.After(t) {
			t = *itm.UpdatedParsed
		}
	}
	return t
}

func (ff *feedFollower) validateSiteInfo(si *SiteInfo) error {
	if _, err := url.Parse(si.FeedUrl); err != nil {
		return err
	}
	if err := shared.ValidateHandle(si.ParrotHandle); err != nil {
		return err
	}
	return nil
}

func (ff *feedFollower) getSiteInfo(urlStr string) (*SiteInfo, *gofeed.Feed, error) {

	urlStr = strings.TrimRight(urlStr, "/")
	var res SiteInfo
	var err error

	// First, let's see if this is the feed itself
	var feed *gofeed.Feed
	var noQueryUrlStr string
	if noQueryUrlStr, err = ff.trimQueryParamsStr(urlStr); err != nil {
		return nil, nil, err
	}
	feed, err = ff.fetchParseFeed(noQueryUrlStr)
	if err == nil {
		res.FeedUrl = noQueryUrlStr
		res.LastUpdated = getLastUpdated(feed)
		res.Title = feed.Title
		res.Description = feed.Description
		res.Url = feed.Link
		res.ParrotHandle = shared.GetHandleFromUrl(res.Url)
		return &res, feed, nil
	}

	// Normalize URL
	siteUrl, err := url.Parse(urlStr)
	if err != nil {
		return nil, nil, err
	}
	res.Url = urlStr
	res.ParrotHandle = shared.GetHandleFromUrl(res.Url)

	// Get the page
	client := http.Client{}
	client.Timeout = feedOrSiteTimeoutSec * time.Second
	resp, err := client.Get(urlStr)
	if err != nil {
		ff.logger.Warnf("Failed to get %s: %v", siteUrl, err)
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("request for %s failed with status %d", siteUrl, resp.StatusCode)
		ff.logger.Warnf("Failed to get site: %v", err)
		return nil, nil, err
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		ff.logger.Warnf("Failed to parse HTML from %s: %v", siteUrl, err)
		return nil, nil, err
	}

	// Pick out the data we're interested in
	res.FeedUrl = ff.getFeedUrl(siteUrl, doc)
	if res.FeedUrl == "" {
		ff.logger.Warnf("No feed URL found: %s", siteUrl)
		return nil, nil, fmt.Errorf("no feed URL found at %s", siteUrl)
	}
	ff.getMetas(doc, &res)

	// Get the feed to make sure it's there, and know when it's last changed
	feed, err = ff.fetchParseFeed(res.FeedUrl)
	if err != nil {
		ff.logger.Warnf("Failed to retrieve and parse feed: %s, %v", res.FeedUrl, err)
		return nil, nil, err
	}
	res.LastUpdated = getLastUpdated(feed)

	return &res, feed, nil
}

func getItemHash(itm *gofeed.Item) uint {
	str := itm.GUID + "\t" + itm.Link
	hasher := murmur3.New32()
	_, _ = hasher.Write([]byte(str))
	return uint(hasher.Sum32())
}

func (ff *feedFollower) updateAccountPosts(
	accountId int,
	accountHandle string,
	feed *gofeed.Feed,
	tootNew bool,
) (err error) {
	err = nil
	var lastKnownFeedUpdated time.Time

	if lastKnownFeedUpdated, err = ff.repo.GetFeedLastUpdated(accountId); err != nil {
		return
	}

	// Deal with feed items newer than our last seen
	// This goes from older to newer
	keepers, newLastUpdated := getSortedPosts(feed.Items, lastKnownFeedUpdated)
	for _, k := range keepers {
		fixPodcastLink(k.itm)
		if err = ff.storePostIfNew(accountId, accountHandle, k.postTime, k.itm, tootNew); err != nil {
			return
		}
	}

	nextCheckDue := ff.getNextCheckTime(newLastUpdated)
	if err = ff.repo.UpdateAccountFeedTimes(accountId, newLastUpdated, nextCheckDue); err != nil {
		return
	}
	return
}

func fixPodcastLink(itm *gofeed.Item) {
	if itm.Link != "" {
		return
	}
	for _, enc := range itm.Enclosures {
		if !strings.HasPrefix(enc.Type, "audio/") || enc.URL == "" {
			continue
		}
		parsedUrl, err := url.Parse(enc.URL)
		if err != nil {
			continue
		}
		parsedUrl.RawQuery = ""
		itm.Link = parsedUrl.String()
		return
	}
}

type sortedPost struct {
	itm      *gofeed.Item
	postTime time.Time
}

func getSortedPosts(items []*gofeed.Item, lastKnownFeedUpdated time.Time) ([]sortedPost, time.Time) {
	var keepers []sortedPost
	newLastUpdated := lastKnownFeedUpdated

	for _, itm := range items {
		keeper, postTime := checkItemTime(itm, lastKnownFeedUpdated)
		if !keeper {
			continue
		}
		if postTime.After(newLastUpdated) {
			newLastUpdated = postTime
		}
		keepers = append(keepers, sortedPost{itm, postTime})
	}

	sort.Slice(keepers, func(i, j int) bool {
		return keepers[i].postTime.Before(keepers[j].postTime)
	})

	return keepers, newLastUpdated
}

func checkItemTime(itm *gofeed.Item, latestKown time.Time) (keeper bool, postTime time.Time) {
	keeper = false
	postTime = time.Time{}
	if itm.PublishedParsed != nil && itm.PublishedParsed.After(latestKown) {
		keeper = true
		postTime = *itm.PublishedParsed
	}
	if itm.UpdatedParsed != nil && itm.UpdatedParsed.After(latestKown) {
		keeper = true
		if itm.UpdatedParsed.After(postTime) {
			postTime = *itm.UpdatedParsed
		}
	}
	return
}

func (ff *feedFollower) getNextCheckTime(lastChanged time.Time) time.Time {

	// Active in the last day: 1 hour
	// Active in the last week: 3 hours
	// Active in the last 4 weeks: 6 hours
	// Inactive for over 4 weeks: 12 hours
	var hours = float64(ff.cfg.UpdateSchedule.Day)
	idleFor := time.Now().Sub(lastChanged)
	if idleFor.Hours() > 24 {
		hours = float64(ff.cfg.UpdateSchedule.Week)
	}
	if idleFor.Hours() > 168 {
		hours = float64(ff.cfg.UpdateSchedule.Weeks4)
	}
	if idleFor.Hours() > 168*4 {
		hours = float64(ff.cfg.UpdateSchedule.Older)
	}

	hours = hours * (0.8 + 0.4*rand.Float64()) // 0.8 - 1.2 random band around targeted value
	res := time.Now().Add(time.Duration(float64(time.Hour) * hours))
	return res
}

func stripHtml(htm string) string {
	p := bluemonday.StrictPolicy()
	plain := p.Sanitize(htm)
	plain = html.UnescapeString(plain)
	plain = strings.TrimSpace(plain)
	return plain
}

func (ff *feedFollower) storePostIfNew(
	accountId int,
	accountHandle string,
	postTime time.Time,
	itm *gofeed.Item,
	tootNew bool,
) (err error) {
	var isNew bool
	plainTitle := stripHtml(itm.Title)
	plainDescription := stripHtml(itm.Description)
	isNew, err = ff.repo.AddFeedPostIfNew(accountId, &dal.FeedPost{
		PostGuidHash: int64(getItemHash(itm)),
		PostTime:     postTime,
		Link:         itm.Link,
		Title:        plainTitle,
		Description:  plainDescription,
	})
	if err != nil {
		return
	}
	if isNew {
		ff.metrics.NewPostSaved()
		if err = ff.createToot(accountId, accountHandle, itm, tootNew); err != nil {
			return
		}
	}
	return
}

func (ff *feedFollower) createToot(accountId int, accountHandle string, itm *gofeed.Item, sendToot bool) error {
	prettyUrl := itm.Link
	prettyUrl = strings.TrimPrefix(prettyUrl, "http://")
	prettyUrl = strings.TrimPrefix(prettyUrl, "https://")
	prettyUrl = strings.TrimRight(prettyUrl, "/")
	plainTitle := stripHtml(itm.Title)
	plainDescription := stripHtml(itm.Description)
	plainDescription = shared.TruncateWithEllipsis(plainDescription, shared.MaxDescriptionLen)
	content := ff.txt.WithVals("toot_new_post.html", map[string]string{
		"title":       plainTitle,
		"url":         itm.Link,
		"prettyUrl":   prettyUrl,
		"description": plainDescription,
	})
	idb := shared.IdBuilder{ff.cfg.Host}
	id := ff.repo.GetNextId()
	statusId := idb.UserStatus(accountHandle, id)
	tootedAt := time.Now()
	err := ff.repo.AddToot(accountId, &dal.Toot{
		PostGuidHash: int64(getItemHash(itm)),
		TootedAt:     tootedAt,
		StatusId:     statusId,
		Content:      content,
	})
	if err != nil {
		return err
	}
	if sendToot {
		if err = ff.messenger.EnqueueBroadcast(accountHandle, statusId, tootedAt, content); err != nil {
			return err
		}
	}
	return nil
}

func (ff *feedFollower) filterFeed(feedUrl string, feed *gofeed.Feed) (FeedStatus, error) {

	// We don't parrot Mastond RSS feeds
	generator := strings.ToLower(feed.Generator)
	if strings.Contains(generator, "mastodon") {
		return FsMastodon, nil
	}

	// We don't parrot blocked feeds
	var err error
	var blocked bool
	blocked, err = ff.blockedFeeds.IsBlocked(feedUrl)
	if err != nil {
		return FsError, err
	}
	if blocked {
		return FsOptOut, nil
	}

	// FsError is the OK response
	return FsError, nil
}

func (ff *feedFollower) GetAccountForFeed(urlStr string) (acct *dal.Account, status FeedStatus, err error) {

	ff.logger.Infof("Retrieving site information: %s", urlStr)

	feedLabel := "failed"
	defer func() {
		ff.metrics.FeedRequested(feedLabel)
	}()

	acct = nil
	status = FsError
	err = nil

	si, feed, siErr := ff.getSiteInfo(urlStr)
	if siErr == nil {
		siErr = ff.validateSiteInfo(si)
	}
	if siErr != nil {
		err = siErr
		return
	}

	status, err = ff.filterFeed(si.FeedUrl, feed)
	if err != nil {
		status = FsError
		return
	}
	if status != FsError {
		return
	}

	idb := shared.IdBuilder{ff.cfg.Host}

	var pubKey string
	var privKey string
	pubKey, privKey, err = ff.keyStore.MakeKeyPair()
	if err != nil {
		ff.logger.Errorf("Failed to create key pair: %v", err)
		return
	}

	var isNew bool
	isNew, err = ff.repo.AddAccountIfNotExist(&dal.Account{
		CreatedAt:   time.Now(),
		Handle:      si.ParrotHandle,
		UserUrl:     idb.UserUrl(si.ParrotHandle),
		FeedName:    si.Title,
		FeedSummary: si.Description,
		SiteUrl:     si.Url,
		FeedUrl:     si.FeedUrl,
		PubKey:      pubKey,
	}, privKey)

	if err != nil {
		ff.logger.Errorf("Failed to create/get account for %s: %v", si.ParrotHandle, isNew)
		return
	}

	ff.logger.Infof("Account is %s; newly created: %v", si.ParrotHandle, isNew)

	acct, err = ff.repo.GetAccount(si.ParrotHandle)
	if err != nil {
		ff.logger.Errorf("Failed to load account for %s; was newly created: %v", si.ParrotHandle, isNew)
		acct = nil
		return
	}

	err = ff.updateAccountPosts(acct.Id, si.ParrotHandle, feed, !isNew)
	if err != nil {
		ff.logger.Errorf("Failed to update account's posts: %s: %v", acct.Handle, err)
		acct = nil
		return
	}

	if isNew {
		status = FsNew
		feedLabel = "new"
	} else {
		status = FsAlreadyFollowed
		feedLabel = "existing"
	}
	return
}

func (ff *feedFollower) fetchParseFeed(feedUrl string) (feed *gofeed.Feed, err error) {

	var req *http.Request
	if req, err = http.NewRequest("GET", feedUrl, nil); err != nil {
		return nil, err
	}
	ff.userAgent.AddUserAgent(req)

	client := http.Client{}
	client.Timeout = time.Second * feedOrSiteTimeoutSec
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %v", resp.StatusCode)
	}

	fp := gofeed.NewParser()
	return fp.Parse(resp.Body)
}

func (ff *feedFollower) updateFeed(acct *dal.Account) error {

	var err error
	ff.logger.Infof("Updating account %s: %s", acct.Handle, acct.FeedUrl)
	ff.metrics.FeedUpdated()

	var feed *gofeed.Feed
	if feed, err = ff.fetchParseFeed(acct.FeedUrl); err != nil {
		return err
	}

	if err = ff.updateAccountPosts(acct.Id, acct.Handle, feed, true); err != nil {
		return err
	}

	go func() {
		if err = ff.PurgeOldPosts(acct, ff.cfg.PostsMinCountKept, ff.cfg.PostsMinDaysKept); err != nil {
			// If purging errors out: swallow it (updateFeed still succeeds); just log
			ff.logger.Errorf("Error purging old posts for account %s: %v", acct.Handle, err)
		}
	}()

	return nil
}

func (ff *feedFollower) PurgeOldPosts(acct *dal.Account, minCount, minAgeDays int) error {

	if minCount <= 0 || minAgeDays <= 0 {
		return nil
	}

	// We're fired off as a goroutine each time a feed has been refreshed
	// Only run one purge at a time
	canProceed := false
	ff.muDeleting.Lock()
	if !ff.isDeleting {
		canProceed = true
		ff.isDeleting = true
	}
	ff.muDeleting.Unlock()
	if !canProceed {
		return nil
	}
	signalDone := func() {
		ff.muDeleting.Lock()
		ff.isDeleting = false
		ff.muDeleting.Unlock()
	}
	defer signalDone()
	time.Sleep(20 * time.Second)

	var err error
	var posts []*dal.FeedPost
	if posts, err = ff.repo.GetPostsExtract(acct.Id); err != nil {
		return err
	}
	// Fewer than minimum count - nothing to do
	if len(posts) <= minCount {
		return nil
	}
	// Sort from newest to oldest
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].PostTime.After(posts[j].PostTime)
	})
	// Check each item from start; if we're past minimum count, mark old enough posts for deletion
	var hashesToDel []int64
	now := time.Now().UTC()
	for i, post := range posts {
		if i < minCount {
			continue
		}
		postAgeDays := now.Sub(post.PostTime).Hours() / 24.0
		if postAgeDays < float64(minAgeDays) {
			continue
		}
		hashesToDel = append(hashesToDel, post.PostGuidHash)
		if len(hashesToDel) == postPurgeBatchSize {
			break
		}
	}
	if len(hashesToDel) == 0 {
		return nil
	}

	// Purge 'em
	ff.logger.Infof("Purging %d old posts from account %s", len(hashesToDel), acct.Handle)
	if err = ff.repo.PurgePostsAndToots(acct.Id, hashesToDel); err != nil {
		return err
	}
	return nil
}

func (ff *feedFollower) purgeUnfollowedAccount(acct *dal.Account) {

	// We're fired off as a goroutine each time there's a deletable account
	// Only run one account deleting at a time
	canProceed := false
	ff.muDeleting.Lock()
	if !ff.isDeleting {
		canProceed = true
		ff.isDeleting = true
	}
	ff.muDeleting.Unlock()
	if !canProceed {
		return
	}
	signalDone := func() {
		ff.muDeleting.Lock()
		ff.isDeleting = false
		ff.muDeleting.Unlock()
	}
	defer signalDone()

	followerCount, err := ff.repo.GetFollowerCount(acct.Handle, false)
	if err != nil {
		ff.logger.Errorf("Error getting follower count of feed: %s: %v", acct.Handle, err)
		return
	}
	if followerCount != 0 {
		return
	}
	ff.logger.Infof("Deleting account with 0 followers: %s", acct.Handle)
	if err = ff.repo.BruteDeleteAccount(acct.Id); err != nil {
		ff.logger.Errorf("Failed to brute-delete account: %s: %v", acct.Handle, err)
		return
	}
	time.Sleep(20 * time.Second)
}

func (ff *feedFollower) updateDBSizeMetric() {

	// In case feed follower is running on a mock config in a unit test: don't bother
	if ff.cfg.DbFile == "" {
		return
	}

	var err error
	var fi os.FileInfo
	fi, err = os.Stat(ff.cfg.DbFile)
	if err != nil {
		ff.logger.Errorf("Error getting DB file size: %v", err)
		return
	}
	ff.metrics.DbFileSize(fi.Size())
}

func (ff *feedFollower) feedCheckLoop() {
	for {
		// This is why we're here
		ff.feedCheckLoopInner()

		// This is real doggone ugly here, but -
		// Other option is to create a logic class just for this
		// Rather a little ugliness here, then all that boilerplate
		// And we're already also setting the "feed-followers" metrics in this module
		ff.updateDBSizeMetric()
	}
}

func (ff *feedFollower) feedCheckLoopInner() {

	defer func() {
		if r := recover(); r != nil {
			const panicSleepSec = 10
			ff.logger.Errorf("Feed check cycle panicked: %v", r)
			ff.logger.Infof("Sleeping %d seconds after panic", panicSleepSec)
			time.Sleep(time.Second * panicSleepSec)
		}
	}()

	var err error
	var acct *dal.Account
	var total int
	if acct, total, err = ff.repo.GetAccountToCheck(time.Now()); err != nil {
		ff.logger.Errorf("Failed to get next feed due for checking: %v", err)
		time.Sleep(feedCheckLoopIdleWakeSec * time.Second)
		return
	}
	ff.metrics.CheckableFeedCount(total)
	if acct == nil {
		ff.logger.Debugf("No feeds to check; sleeping %d seconds", feedCheckLoopIdleWakeSec)
		time.Sleep(feedCheckLoopIdleWakeSec * time.Second)
		return
	}
	lastUpdated := acct.FeedLastUpdated
	err = ff.updateFeed(acct)
	if err != nil {
		ff.logger.Errorf("Error updating feed: %s: %v", acct.Handle, err)
		// Reschedule for updating as if there was no new post
		nextCheckDue := ff.getNextCheckTime(lastUpdated)
		if err = ff.repo.UpdateAccountFeedTimes(acct.Id, lastUpdated, nextCheckDue); err != nil {
			ff.logger.Errorf("Failed to reschedule for checking after error: %s: %v", acct.Handle, err)
		}
	}
	// If no error, updateFeed has set next due date for checking
	// Delete account if no followers; purge old posts
	go ff.purgeUnfollowedAccount(acct)
}
