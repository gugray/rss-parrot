package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"rss_parrot/dal"
	"rss_parrot/shared"
	"strconv"
	"strings"
	"time"
)

const tmplPathPrefx = "www/"
const versionFileName = "version.txt"
const feedsPerPage = 200

var months = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

type webHandlerGroup struct {
	cfg           *shared.Config
	logger        shared.ILogger
	repo          dal.IRepo
	version       string
	timestamp     string
	pageTemplates map[string]*template.Template
}

func NewWebHandlerGroup(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
) IHandlerGroup {
	res := webHandlerGroup{
		cfg:           cfg,
		logger:        logger,
		repo:          repo,
		timestamp:     fmt.Sprintf("%d", time.Now().UnixMilli()),
		pageTemplates: make(map[string]*template.Template),
	}
	versionBytes, _ := os.ReadFile(tmplPathPrefx + versionFileName)
	res.version = string(versionBytes)
	res.initTemplates()
	return &res
}

func (hg *webHandlerGroup) Prefix() string {
	return "/web"
}

func (hg *webHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"GET", "/feeds", func(w http.ResponseWriter, r *http.Request) { hg.getFeeds(w, r) }},
		{"GET", "/about", func(w http.ResponseWriter, r *http.Request) { hg.getAbout(w, r) }},
		{"GET", rootPlacholder, func(w http.ResponseWriter, r *http.Request) { hg.getRoot(w, r) }},
	}
}

func (hg *webHandlerGroup) AuthMW() func(next http.Handler) http.Handler {
	return emptyMW
}

func (hg *webHandlerGroup) initTemplates() {
	mainFiles, err := filepath.Glob(tmplPathPrefx + "main-*.tmpl")
	if err != nil {
		hg.logger.Errorf("Failed to list main-*.tmpl: %v", err)
		panic(err)
	}
	for _, fn := range mainFiles {
		mainName := strings.TrimPrefix(fn, tmplPathPrefx+"main-")
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

	idb := shared.IdBuilder{hg.cfg.Host}
	addTemplateFuncs(t, &idb)

	var err error
	var tmplFiles []string
	foundMain := false
	if tmplFiles, err = filepath.Glob(tmplPathPrefx + "*.tmpl"); err != nil {
		return nil, err
	}
	for _, fn := range tmplFiles {
		include := true
		if strings.HasPrefix(fn, tmplPathPrefx+"main-") {
			if fn == tmplPathPrefx+"main-"+mainName+".tmpl" {
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

func addTemplateFuncs(t *template.Template, idb *shared.IdBuilder) {

	profileUrl := func(handle string) string {
		return idb.UserProfile(handle)
	}

	prettyDate := func(t time.Time) string {
		return fmt.Sprintf("%s %d, %d", months[t.Month()-1], t.Day(), t.Year())
	}

	t.Funcs(template.FuncMap{
		"isNonEmptyString": func(s string) bool { return s != "" },
		"prettyDate":       prettyDate,
		"profileUrl":       profileUrl,
	})
}

type baseModel struct {
	Timestamp     string
	Version       string
	LnkFeedsClass string
	LnkAboutClass string
	Data          any
}

func (hg *webHandlerGroup) getRoot(w http.ResponseWriter, r *http.Request) {

	t, model := hg.mustGetPageTemplate("root")
	t.ExecuteTemplate(w, "index.tmpl", model)
}

func (hg *webHandlerGroup) getAbout(w http.ResponseWriter, r *http.Request) {

	t, model := hg.mustGetPageTemplate("about")
	model.LnkAboutClass = "selected"
	t.ExecuteTemplate(w, "index.tmpl", model)
}

type feedModel struct {
	Feeds []*dal.Account
	Pages []pageLink
}

type pageLink struct {
	Query   string
	Display int
	Class   string
}

func (hg *webHandlerGroup) getFeeds(w http.ResponseWriter, r *http.Request) {

	var err error
	pageIx := 0
	pageParam := r.URL.Query().Get("page")
	if pageIx, err = strconv.Atoi(pageParam); err != nil || pageIx < 0 {
		pageIx = 0
	}

	var accounts []*dal.Account
	var total int
	accounts, total, err = hg.repo.GetAccountsPage(pageIx*feedsPerPage, feedsPerPage)
	// TODO: bespoke error if query failed
	if err != nil {
		accounts = make([]*dal.Account, 0)
		total = 0
	}

	data := feedModel{
		Feeds: accounts,
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
	for _, a := range data.Feeds {
		a.Name = strings.TrimPrefix(a.Name, "🦜 ")
	}

	t, model := hg.mustGetPageTemplate("feeds")
	model.LnkFeedsClass = "selected"
	model.Data = &data

	w.Header().Set("X-Robots-Tag", "noindex")
	t.ExecuteTemplate(w, "index.tmpl", model)
}
