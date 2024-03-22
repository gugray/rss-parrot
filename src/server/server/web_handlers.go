package server

import (
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"rss_parrot/dal"
	"rss_parrot/logic"
	"rss_parrot/shared"
	"rss_parrot/texts"
	"strconv"
	"strings"
	"time"
)

const versionFileName = "version.txt"
const feedsPerPage = 200
const postsPerPage = 100

var months = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

type webHandlerGroup struct {
	cfg           *shared.Config
	logger        shared.ILogger
	repo          dal.IRepo
	txt           texts.ITexts
	metrics       logic.IMetrics
	idb           shared.IdBuilder
	version       string
	timestamp     string
	pageTemplates map[string]*template.Template
}

func NewWebHandlerGroup(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	txt texts.ITexts,
	metrics logic.IMetrics,
) IHandlerGroup {
	res := webHandlerGroup{
		cfg:           cfg,
		logger:        logger,
		repo:          repo,
		txt:           txt,
		metrics:       metrics,
		idb:           shared.IdBuilder{cfg.Host},
		timestamp:     fmt.Sprintf("%d", time.Now().UnixMilli()),
		pageTemplates: make(map[string]*template.Template),
	}
	versionBytes, _ := os.ReadFile(wwwPathPrefx + versionFileName)
	res.version = string(versionBytes)
	res.initTemplates()
	return &res
}

func (hg *webHandlerGroup) Prefix() string {
	return "/web"
}

func (hg *webHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"GET", "/feeds/{feed}", func(w http.ResponseWriter, r *http.Request) { hg.getOneFeed(w, r) }},
		{"GET", "/feeds", func(w http.ResponseWriter, r *http.Request) { hg.getFeeds(w, r) }},
		{"GET", "/changes", func(w http.ResponseWriter, r *http.Request) { hg.getChanges(w, r) }},
		{"GET", "/about", func(w http.ResponseWriter, r *http.Request) { hg.getAbout(w, r) }},
		{"GET", rootPlacholder, func(w http.ResponseWriter, r *http.Request) { hg.getRoot(w, r) }},
		{"GET", notFoundPlacholder, func(w http.ResponseWriter, r *http.Request) { hg.send404(w, r) }},
	}
}

func (hg *webHandlerGroup) AuthMW() func(next http.Handler) http.Handler {
	return emptyMW
}

func (hg *webHandlerGroup) initTemplates() {
	mainFiles, err := filepath.Glob(wwwPathPrefx + "main-*.tmpl")
	if err != nil {
		hg.logger.Errorf("Failed to list main-*.tmpl: %v", err)
		panic(err)
	}
	for _, fn := range mainFiles {
		mainName := strings.TrimPrefix(fn, wwwPathPrefx+"main-")
		mainName = strings.TrimSuffix(mainName, ".tmpl")
		var t *template.Template
		if t, err = hg.parsePageTemplate(mainName); err != nil {
			hg.logger.Errorf("Failed to parse page template: %s: %v", fn, err)
			panic(err)
		}
		hg.pageTemplates[mainName] = t
	}
}
func (hg *webHandlerGroup) parsePageTemplate(mainName string) (*template.Template, error) {

	t := template.New("master")

	hg.addTemplateFuncs(t)

	var err error
	var tmplFiles []string
	foundMain := false
	if tmplFiles, err = filepath.Glob(wwwPathPrefx + "*.tmpl"); err != nil {
		return nil, err
	}
	for _, fn := range tmplFiles {
		include := true
		if strings.HasPrefix(fn, wwwPathPrefx+"main-") {
			if fn == wwwPathPrefx+"main-"+mainName+".tmpl" {
				foundMain = true
			} else {
				include = false
			}
		}
		if !include {
			continue
		}
		if _, err = t.ParseFiles(fn); err != nil {
			return nil, err
		}
	}
	if !foundMain {
		err = fmt.Errorf("did not find 'main' template for %s", mainName)
		return nil, err
	}
	return t, nil
}

func (hg *webHandlerGroup) mustGetPageTemplate(mainName string) (*template.Template, *baseModel) {
	bm := baseModel{Version: hg.version}
	if hg.cfg.CachePageTemplates {
		t, found := hg.pageTemplates[mainName]
		if !found {
			err := fmt.Errorf("Page template not found: %s", mainName)
			hg.logger.Errorf("%v", err)
			panic(err)
		}
		bm.Timestamp = hg.timestamp
		return t, &bm
	} else {
		t, err := hg.parsePageTemplate(mainName)
		if err != nil {
			hg.logger.Errorf("%v", err)
			panic(err)
		}
		bm.Timestamp = fmt.Sprintf("%d", time.Now().UnixMilli())
		return t, &bm
	}
}

func (hg *webHandlerGroup) addTemplateFuncs(t *template.Template) {

	profileUrl := func(handle string) string {
		return hg.idb.UserProfile(handle)
	}
	prettyDate := func(t time.Time) string {
		return fmt.Sprintf("%s %d, %d", months[t.Month()-1], t.Day(), t.Year())
	}
	prettyDateTime := func(t time.Time) string {
		dateStr := prettyDate(t)
		return fmt.Sprintf("%s %02d:%02d", dateStr, t.Hour(), t.Minute())
	}

	t.Funcs(template.FuncMap{
		"isNonEmptyString": func(s string) bool { return s != "" },
		"prettyDate":       prettyDate,
		"prettyDateTime":   prettyDateTime,
		"profileUrl":       profileUrl,
	})
}

type baseModel struct {
	Timestamp       string
	Version         string
	LnkFeedsClass   string
	LnkChangesClass string
	LnkAboutClass   string
	Data            any
}

func (hg *webHandlerGroup) getRoot(w http.ResponseWriter, r *http.Request) {

	obs := hg.metrics.StartWebRequestIn(r.URL.Path)
	defer obs.Finish()

	t, model := hg.mustGetPageTemplate("root")
	t.ExecuteTemplate(w, "index.tmpl", model)
}

func (hg *webHandlerGroup) send404(w http.ResponseWriter, r *http.Request) {

	obs := hg.metrics.StartWebRequestIn("/404")
	defer obs.Finish()

	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != "GET" || acceptsJson(r) {
		writeErrorResponse(w, notFoundStr, http.StatusNotFound)
		return
	}

	t, model := hg.mustGetPageTemplate("404")
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("X-Robots-Tag", "noindex")
	t.ExecuteTemplate(w, "index.tmpl", model)
}

func (hg *webHandlerGroup) send500(w http.ResponseWriter, r *http.Request) {

	obs := hg.metrics.StartWebRequestIn("/500")
	defer obs.Finish()

	t, model := hg.mustGetPageTemplate("500")
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("X-Robots-Tag", "noindex")
	t.ExecuteTemplate(w, "index.tmpl", model)
}

func (hg *webHandlerGroup) getAbout(w http.ResponseWriter, r *http.Request) {

	obs := hg.metrics.StartWebRequestIn(r.URL.Path)
	defer obs.Finish()

	t, model := hg.mustGetPageTemplate("about")
	model.LnkAboutClass = "selected"
	t.ExecuteTemplate(w, "index.tmpl", model)
}

func (hg *webHandlerGroup) getChanges(w http.ResponseWriter, r *http.Request) {

	obs := hg.metrics.StartWebRequestIn(r.URL.Path)
	defer obs.Finish()

	t, model := hg.mustGetPageTemplate("changes")
	model.LnkChangesClass = "selected"
	t.ExecuteTemplate(w, "index.tmpl", model)
}

type feedsModel struct {
	Feeds []*dal.Account
	Pages []pageLink
}

type pageLink struct {
	Query   string
	Display int
	Class   string
}

func (hg *webHandlerGroup) getFeeds(w http.ResponseWriter, r *http.Request) {

	obs := hg.metrics.StartWebRequestIn(r.URL.Path)
	defer obs.Finish()

	var err error
	pageIx := 0
	pageParam := r.URL.Query().Get("page")
	if pageIx, err = strconv.Atoi(pageParam); err != nil || pageIx < 0 {
		pageIx = 0
	}

	var accounts []*dal.Account
	var total int
	accounts, total, err = hg.repo.GetAccountsPage(pageIx*feedsPerPage, feedsPerPage)

	if err != nil {
		hg.logger.Errorf("Error retrieving feeds %v", err)
		hg.send500(w, r)
		return
	}

	// Remove birb's built-in account
	total -= 1
	nonBuiltInAccounts := make([]*dal.Account, 0, len(accounts))
	for _, acct := range accounts {
		if acct.Handle == hg.cfg.Birb.User {
			continue
		}
		nonBuiltInAccounts = append(nonBuiltInAccounts, acct)
	}

	data := feedsModel{
		Feeds: nonBuiltInAccounts,
	}
	for i := 0; i < total/feedsPerPage+1; i++ {
		pl := pageLink{
			Display: i + 1,
		}
		if i == pageIx {
			pl.Class = "selected"
		}
		if i != 0 {
			pl.Query = fmt.Sprintf("?page=%d", i)
		}
		data.Pages = append(data.Pages, pl)
	}

	t, model := hg.mustGetPageTemplate("feeds")
	model.LnkFeedsClass = "selected"
	model.Data = &data

	w.Header().Set("X-Robots-Tag", "noindex")
	t.ExecuteTemplate(w, "index.tmpl", model)
}

type oneFeedModel struct {
	Handle          string
	Name            string
	Bio             template.HTML
	SiteUrl         string
	SiteUrlNoSchema string
	FeedUrl         string
	FeedUrlNoSchema string
	FollowerCount   uint
	PostCount       uint
	Posts           []*dal.FeedPost
	NotShownPosts   uint
}

func (hg *webHandlerGroup) loadFeedData(acct *dal.Account) *oneFeedModel {

	var err error

	bio := hg.txt.WithVals("acct_bio.html", map[string]string{
		"siteUrl":     hg.idb.SiteUrl(),
		"description": acct.FeedSummary,
	})
	var followerCount, postCount uint
	if followerCount, err = hg.repo.GetApprovedFollowerCount(acct.Handle); err != nil {
		hg.logger.Errorf("Error retrieving follower count for %s: %v", acct.Handle, err)
		return nil
	}
	if postCount, err = hg.repo.GetPostCount(acct.Handle); err != nil {
		hg.logger.Errorf("Error retrieving post count for %s: %v", acct.Handle, err)
		return nil
	}

	data := oneFeedModel{
		Handle:        shared.MakeFullMoniker(hg.cfg.Host, acct.Handle),
		Name:          shared.GetNameWithParrot(acct.FeedName),
		Bio:           template.HTML(bio),
		SiteUrl:       acct.SiteUrl,
		FeedUrl:       acct.FeedUrl,
		FollowerCount: followerCount,
		PostCount:     postCount,
	}
	data.FeedUrlNoSchema = strings.TrimPrefix(data.FeedUrl, "https://")
	data.FeedUrlNoSchema = strings.TrimPrefix(data.FeedUrlNoSchema, "http://")
	data.SiteUrlNoSchema = strings.TrimPrefix(data.SiteUrl, "https://")
	data.SiteUrlNoSchema = strings.TrimPrefix(data.SiteUrlNoSchema, "http://")

	data.Posts, err = hg.repo.GetPostsPage(acct.Id, 0, postsPerPage)
	if err != nil {
		hg.logger.Errorf("Error retrieving posts for %s: %v", acct.Handle, err)
		return nil
	}

	for _, p := range data.Posts {
		p.Description = shared.TruncateWithEllipsis(p.Description, shared.MaxDescriptionLen)
	}

	if data.PostCount > uint(len(data.Posts)) {
		data.NotShownPosts = data.PostCount - uint(len(data.Posts))
	}

	return &data
}

func (hg *webHandlerGroup) getOneFeed(w http.ResponseWriter, r *http.Request) {

	obs := hg.metrics.StartWebRequestIn("/feeds/<feed>")
	defer obs.Finish()

	hg.logger.Infof("Handling user GET: %s", r.URL.Path)
	feedName := mux.Vars(r)["feed"]
	feedName = strings.ToLower(feedName)

	if feedName == hg.cfg.Birb.User {
		hg.logger.Infof("Requesting profile of '%s'; redirecting to root", hg.cfg.Birb.User)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	acct, err := hg.repo.GetAccount(feedName)
	if acct == nil {
		if err == nil {
			hg.logger.Infof("Feed '%s' doesn't exist; returning a 404", feedName)
			hg.send404(w, r)
			return
		} else {
			hg.logger.Errorf("Error retrieving feed %s: %v", feedName, err)
			hg.send500(w, r)
			return
		}
	}

	data := hg.loadFeedData(acct)
	if data == nil {
		hg.send500(w, r)
		return
	}

	t, model := hg.mustGetPageTemplate("one-feed")
	model.LnkFeedsClass = "selected"
	model.Data = data

	w.Header().Set("X-Robots-Tag", "noindex")
	t.ExecuteTemplate(w, "index.tmpl", model)
}
