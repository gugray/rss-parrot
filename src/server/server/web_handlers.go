package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"rss_parrot/shared"
	"strings"
	"time"
)

const tmplPathPrefx = "www/"
const versionFileName = "version.txt"

type webHandlerGroup struct {
	cfg           *shared.Config
	logger        shared.ILogger
	version       string
	timestamp     string
	pageTemplates map[string]*template.Template
}

func NewWebHandlerGroup(
	cfg *shared.Config,
	logger shared.ILogger,
) IHandlerGroup {
	res := webHandlerGroup{
		cfg:           cfg,
		logger:        logger,
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
	var err error
	var tmplFiles []string
	if tmplFiles, err = filepath.Glob(tmplPathPrefx + "*.tmpl"); err != nil {
		return nil, err
	}
	for _, fn := range tmplFiles {
		include := true
		if strings.HasPrefix(fn, tmplPathPrefx+"main-") {
			if fn != tmplPathPrefx+"main-"+mainName+".tmpl" {
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

type baseModel struct {
	Timestamp string
	Version   string
	Data      any
}

func (hg *webHandlerGroup) getRoot(w http.ResponseWriter, r *http.Request) {

	t, model := hg.mustGetPageTemplate("root")
	t.ExecuteTemplate(w, "index.tmpl", model)
	//_, _ = fmt.Fprintln(w, "This is the root of all goodness.")
}

func (hg *webHandlerGroup) getFeeds(w http.ResponseWriter, r *http.Request) {

	_, _ = fmt.Fprintln(w, "Hello, sailor.")
}
