package texts

import (
	"embed"
	"fmt"
	"html"
	"strings"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../test/mocks/mock_texts.go -package mocks rss_parrot/texts ITexts

//go:embed snippets
var fs embed.FS

type ITexts interface {
	Get(id string) string
	WithVals(id string, vals map[string]string) string
}

func NewTexts() ITexts {
	return &texts{}
}

type texts struct {
}

func (t *texts) Get(id string) string {
	fn := fmt.Sprintf("snippets/%s", id)
	bytes, err := fs.ReadFile(fn)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func (t *texts) WithVals(id string, vals map[string]string) string {
	res := t.Get(id)
	isHtml := strings.HasSuffix(id, ".html")
	for ph := range vals {
		pattern := fmt.Sprintf("{{%s}}", ph)
		val := vals[ph]
		if isHtml {
			val = html.EscapeString(val)
		}
		res = strings.ReplaceAll(res, pattern, val)
	}
	return res
}
