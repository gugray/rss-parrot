package writers

import (
	"encoding/json"
	"fmt"
)

func WriteWebfingerJson(baseUrl, userName string) string {
	resp := WebfingerResp{
		Subject: fmt.Sprintf("acct:%s@%s", userName, baseUrl),
		Aliases: nil,
		Links: []WebfingerLink{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: fmt.Sprintf("https://%s/@%s", baseUrl, userName),
			},
		},
	}
	json, _ := json.Marshal(resp)
	return string(json)
}

type WebfingerResp struct {
	Subject string          `json:"subject"`
	Aliases []string        `json:"aliases"`
	Links   []WebfingerLink `json:"links"`
}

type WebfingerLink struct {
	Rel      string `json:"rel"`
	Type     string `json:"type,omitempty"`
	Href     string `json:"href,omitempty"`
	Template string `json:"template,omitempty"`
}
