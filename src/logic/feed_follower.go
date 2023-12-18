package logic

import (
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"rss_parrot/shared"
)

type IFeedFollower interface {
	GetSiteInfo(siteUrl string) *SiteInfo
}

type SiteInfo struct {
	Url         string
	FeedUrl     string
	Title       string
	Description string
}

type feedFollower struct {
	cfg               *shared.Config
	logger            shared.ILogger
	reTitle           regexp.Regexp
	reDesciptionTag   regexp.Regexp
	reDesriptionValue regexp.Regexp
	reFeedTag         regexp.Regexp
	reFeedValue       regexp.Regexp
}

func NewFeedFollower(cfg *shared.Config, logger shared.ILogger) IFeedFollower {
	return &feedFollower{
		cfg:               cfg,
		logger:            logger,
		reTitle:           *regexp.MustCompile(`<title>([^<]+)</title>`),
		reDesciptionTag:   *regexp.MustCompile(`<meta [^>]*name.?=.?['"]?description['"]?[^>]*>`),
		reDesriptionValue: *regexp.MustCompile(`content=("([^"]+)"|'([^']+)')`),
		reFeedTag:         *regexp.MustCompile(`<link [^>]*type=["']?application/(rss|atom)\+xml["']?[^>]*>`),
		reFeedValue:       *regexp.MustCompile(`href=('([^']+)'|"([^"]+)")`),
	}
}

func (ff *feedFollower) getSite(url string) *[]byte {
	var err error
	client := &http.Client{}
	var req *http.Request
	if req, err = http.NewRequest("GET", url, nil); err != nil {
		ff.logger.Infof("Failed to create GET request: %v: %s", err, url)
		return nil
	}
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		ff.logger.Infof("Failed to GET site: %v: %s", err, url)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		ff.logger.Infof("Failed to GET site: status %d: %s", resp.StatusCode, url)
		return nil
	}
	defer resp.Body.Close()
	var bodyBytes []byte
	if bodyBytes, err = io.ReadAll(resp.Body); err != nil {
		ff.logger.Infof("Failed to read response: %v: %s", err, url)
		return nil
	}
	return &bodyBytes
}

func (ff *feedFollower) extractSiteInfo(siteUrlStr, body string) *SiteInfo {

	var res SiteInfo
	res.Url = siteUrlStr

	siteUrl, err := url.Parse(siteUrlStr)
	if err != nil {
		return nil
	}
	feedTag := ff.reFeedTag.FindString(body)
	if feedTag == "" {
		return nil
	}
	groups := ff.reFeedValue.FindStringSubmatch(feedTag)
	if groups == nil {
		return nil
	}
	feedUrlStr := groups[2]
	if feedUrlStr == "" {
		feedUrlStr = groups[3]
	}
	feedUrl, err := url.Parse(feedUrlStr)
	if err != nil {
		return nil
	}
	if !feedUrl.IsAbs() {
		feedUrl = siteUrl.ResolveReference(feedUrl)
	}
	res.FeedUrl = feedUrl.String()

	groups = ff.reTitle.FindStringSubmatch(body)
	if groups != nil {
		res.Title = groups[1]
	}
	descrTag := ff.reDesciptionTag.FindString(body)
	groups = ff.reDesriptionValue.FindStringSubmatch(descrTag)
	if groups != nil {
		res.Description = groups[2]
		if res.Description == "" {
			res.Description = groups[3]
		}
	}
	res.Title = html.UnescapeString(res.Title)
	res.Description = html.UnescapeString(res.Description)

	return &res
}

func (ff *feedFollower) GetSiteInfo(siteUrl string) *SiteInfo {
	bodyBytes := ff.getSite(siteUrl)
	if bodyBytes == nil {
		return nil
	}
	body := string(*bodyBytes)
	return ff.extractSiteInfo(siteUrl, body)

	// TODO: Do we already follow this feed?
	// TODO: Retrieve feed to check it's there!
}
