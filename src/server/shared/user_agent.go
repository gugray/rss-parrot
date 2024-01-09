package shared

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	versionFileName   = "www/version.txt"
	userAgentTemplate = "RSS-Parrot-Bot/%s (+https://%s)"
)

type IUserAgent interface {
	AddUserAgent(req *http.Request)
}

type userAgent struct {
	userAgentValue string
}

func NewUserAgent(cfg *Config) IUserAgent {
	return &userAgent{
		userAgentValue: buildUserAgentString(cfg.Host),
	}
}

func buildUserAgentString(host string) string {
	versionBytes, _ := os.ReadFile(versionFileName)
	versionStr := string(versionBytes)
	versionStr = strings.TrimPrefix(versionStr, "v")
	return fmt.Sprintf(userAgentTemplate, versionStr, host)
}

func (ua *userAgent) AddUserAgent(req *http.Request) {
	req.Header.Add("User-Agent", ua.userAgentValue)
}
