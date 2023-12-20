package logic

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"github.com/twmb/murmur3"
	"math/rand"
	"net/http"
	"net/url"
	"rss_parrot/dal"
	"rss_parrot/shared"
	"rss_parrot/texts"
	"strings"
	"time"
)

type IFeedFollower interface {
	GetAccountForFeed(urlStr string) *dal.Account
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
	cfg        *shared.Config
	logger     shared.ILogger
	repo       dal.IRepo
	txt        texts.ITexts
	keyHandler IKeyHandler
}

func NewFeedFollower(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	txt texts.ITexts,
	keyHandler IKeyHandler,
) IFeedFollower {
	return &feedFollower{
		cfg:        cfg,
		logger:     logger,
		repo:       repo,
		txt:        txt,
		keyHandler: keyHandler,
	}
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
	// It's a keeper
	res := feedUrl.String()
	res = strings.TrimRight(res, "/")
	return res
}

func (ff *feedFollower) getMetas(siteUrl *url.URL, doc *goquery.Document, si *SiteInfo) {
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

func (ff *feedFollower) getSiteInfo(urlStr string) (*SiteInfo, *gofeed.Feed) {

	urlStr = strings.TrimRight(urlStr, "/")
	var res SiteInfo
	var err error

	// First, let's see if this is the feed itself
	fp := gofeed.NewParser()
	var feed *gofeed.Feed
	feed, err = fp.ParseURL(urlStr)
	if err == nil {
		res.FeedUrl = urlStr
		res.LastUpdated = getLastUpdated(feed)
		res.Title = feed.Title
		res.Description = feed.Description
		res.Url = feed.Link
		res.ParrotHandle = shared.GetHandleFromUrl(res.Url)
		return &res, feed
	}

	// Normalize URL
	siteUrl, err := url.Parse(urlStr)
	if err != nil {
		return nil, nil
	}
	res.Url = urlStr
	res.ParrotHandle = shared.GetHandleFromUrl(res.Url)

	// Get the page
	resp, err := http.Get(urlStr)
	if err != nil {
		ff.logger.Warnf("Failed to get %s: %v", siteUrl, err)
		return nil, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		ff.logger.Warnf("Failed to get %s: status %d", siteUrl, resp.StatusCode)
		return nil, nil
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		ff.logger.Warnf("Failed to parse HTML from %s: %v", siteUrl, err)
		return nil, nil
	}

	// Pick out the data we're interested in
	res.FeedUrl = ff.getFeedUrl(siteUrl, doc)
	if res.FeedUrl == "" {
		ff.logger.Warnf("No feed URL found: %s", siteUrl)
		return nil, nil
	}
	ff.getMetas(siteUrl, doc, &res)

	// Get the feed to make sure it's there, and know when it's last changed
	feed, err = fp.ParseURL(res.FeedUrl)
	if err != nil {
		ff.logger.Warnf("Failed to retrieve and parse feed: %s, %v", res.FeedUrl, err)
		return nil, nil
	}
	res.LastUpdated = getLastUpdated(feed)

	return &res, feed
}

func getGuidHash(guid string) int {
	hasher := murmur3.New32()
	hash, _ := hasher.Write([]byte(guid))
	return hash
}

func (ff *feedFollower) updateAccountPosts(accountId int, feed *gofeed.Feed, tootNew bool) (err error) {
	err = nil
	var lastKnownFeedUpdated time.Time

	if lastKnownFeedUpdated, err = ff.repo.GetFeedLastUpdated(accountId); err != nil {
		return
	}

	newLastUpdated := lastKnownFeedUpdated
	// Deal with feed items newer than our last seen
	for _, itm := range feed.Items {
		keeper := false
		var postTime time.Time
		if itm.PublishedParsed != nil && itm.PublishedParsed.After(lastKnownFeedUpdated) {
			keeper = true
			postTime = *itm.PublishedParsed
		}
		if itm.UpdatedParsed != nil && itm.UpdatedParsed.After(lastKnownFeedUpdated) {
			keeper = true
			if itm.UpdatedParsed.After(postTime) {
				postTime = *itm.UpdatedParsed
			}
		}
		if !keeper {
			continue
		}
		if postTime.After(newLastUpdated) {
			newLastUpdated = postTime
		}
		if err = ff.storePostIfNew(accountId, postTime, itm, tootNew); err != nil {
			return
		}
	}

	nextCheckDue := ff.getNextCheckTime(newLastUpdated)
	if err = ff.repo.UpdateAccountFeedTimes(accountId, newLastUpdated, nextCheckDue); err != nil {
		return
	}
	return
}

func (ff *feedFollower) getNextCheckTime(lastChanged time.Time) time.Time {
	res := time.Now()
	// TODO: check less active feeds less frequently
	hours := 6.0 * (0.8 + 0.4*rand.Float64()) // 0.8 - 1.2 random band around targeted value
	res = res.Add(time.Duration(float64(time.Hour) * hours))
	return res
}

func (ff *feedFollower) storePostIfNew(
	accountId int,
	postTime time.Time,
	itm *gofeed.Item,
	tootNew bool,
) (err error) {
	var isNew bool
	isNew, err = ff.repo.AddFeedPostIfNew(accountId, &dal.FeedPost{
		PostGuidHash: getGuidHash(itm.Link),
		PostTime:     postTime,
		Link:         itm.Link,
		Title:        itm.Title,
		Desription:   itm.Description,
	})
	if err != nil {
		return
	}
	if isNew && tootNew {
		if err = ff.createToot(accountId, itm); err != nil {
			return
		}
	}
	return
}

func (ff *feedFollower) createToot(accountId int, itm *gofeed.Item) error {
	// TODO
	return nil
}

func (ff *feedFollower) GetAccountForFeed(urlStr string) *dal.Account {

	ff.logger.Infof("Retrieving site information: %s", urlStr)

	var err error

	si, feed := ff.getSiteInfo(urlStr)
	if si == nil {
		return nil
	}

	idb := shared.IdBuilder{ff.cfg.Host}

	var pubKey string
	var privKey string
	pubKey, privKey, err = ff.keyHandler.MakeKeyPair()
	if err != nil {
		ff.logger.Errorf("Failed to create key pair: %v", err)
		return nil
	}

	isNew, err := ff.repo.AddAccountIfNotExist(&dal.Account{
		CreatedAt: time.Now(),
		Handle:    si.ParrotHandle,
		UserUrl:   idb.UserUrl(si.ParrotHandle),
		Name:      shared.GetNameWithParrot(si.Title),
		Summary:   ff.txt.WithVals("acct_bio.html", map[string]string{"description": si.Description}),
		SiteUrl:   si.Url,
		FeedUrl:   si.FeedUrl,
		PubKey:    pubKey,
	}, privKey)

	if err != nil {
		ff.logger.Errorf("Failed to create/get account for %s: %v", si.ParrotHandle, isNew)
		return nil
	}

	ff.logger.Infof("Account is %s; newly created: %v", si.ParrotHandle, isNew)

	var acct *dal.Account
	acct, err = ff.repo.GetAccount(si.ParrotHandle)
	if err != nil {
		ff.logger.Errorf("Failed to load account for %s; was newly created: %v", si.ParrotHandle, isNew)
		return nil
	}

	err = ff.updateAccountPosts(acct.Id, feed, false)
	if err != nil {
		ff.logger.Errorf("Failed to update account's posts: %s: %v", acct.Handle, err)
		return nil
	}

	return acct
}
