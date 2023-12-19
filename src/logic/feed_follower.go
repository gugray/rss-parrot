package logic

import (
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"rss_parrot/shared"
	"strings"
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

func (ff *feedFollower) getFeedUrl(body string, siteUrl *url.URL) string {
	feedTags := ff.reFeedTag.FindAllString(body, -1)
	res := ""
	resIsRss := false
	for _, feedTag := range feedTags {
		// Some links we find are not even the feeds we want
		// <link rel="service.post" type="application/atom+xml" title="..." />
		if !strings.Contains(feedTag, "alternate") {
			continue
		}
		// If there are multiple feeds, we want the RSS one
		isRss := strings.Contains(feedTag, "rss+xml")
		if res != "" && resIsRss && !isRss {
			continue
		}
		// Get the href value
		groups := ff.reFeedValue.FindStringSubmatch(feedTag)
		if groups == nil {
			continue
		}
		feedUrlStr := groups[2]
		if feedUrlStr == "" {
			feedUrlStr = groups[3]
		}
		// Make it absolute
		feedUrl, err := url.Parse(feedUrlStr)
		if err != nil {
			continue
		}
		if !feedUrl.IsAbs() {
			feedUrl = siteUrl.ResolveReference(feedUrl)
		}
		// It's a keeper
		res = feedUrl.String()
		res = strings.TrimRight(res, "/")
		resIsRss = isRss

	}
	return res
}

func (ff *feedFollower) extractSiteInfo(siteUrlStr, body string) *SiteInfo {

	var res SiteInfo

	siteUrlStr = strings.TrimRight(siteUrlStr, "/")
	res.Url = siteUrlStr

	siteUrl, err := url.Parse(siteUrlStr)
	if err != nil {
		return nil
	}
	res.FeedUrl = ff.getFeedUrl(body, siteUrl)
	if res.FeedUrl == "" {
		return nil
	}

	groups := ff.reTitle.FindStringSubmatch(body)
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
