package shared

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

const ActivityPublic = "https://www.w3.org/ns/activitystreams#Public"

func GetHostName(userUrl string) (string, error) {
	var parsedUrl *url.URL
	var urlError error
	parsedUrl, urlError = url.Parse(userUrl)
	if urlError != nil {
		return "", fmt.Errorf("Failed to parse user URL '%s': %v", userUrl, urlError)
	}
	return parsedUrl.Hostname(), nil
}

func MakeFullMoniker(hostName, handle string) string {
	return "@" + handle + "@" + hostName
}

func GetNameWithParrot(name string) string {
	return "🦜 " + name
}

func TruncateWithEllipsis(text string, maxLen int) string {
	// https://stackoverflow.com/a/73939904/7479498
	lastSpaceIx := maxLen
	len := 0
	for i, r := range text {
		if unicode.IsSpace(r) {
			lastSpaceIx = i
		}
		len++
		if len > maxLen {
			return text[:lastSpaceIx] + "…"
		}
	}
	// If here, string is shorter or equal to maxLen
	return text
}

func GetHandleFromUrl(url string) string {

	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimRight(url, "/")

	var buf bytes.Buffer
	for i := 0; i < len(url); i++ {
		c := url[i]
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '-' || c == '.' {
			buf.WriteByte(c)
		} else {
			buf.WriteString(".")
		}
	}
	res := strings.ToLower(buf.String())

	for {
		merged := strings.ReplaceAll(res, "..", ".")
		if len(merged) == len(res) {
			break
		}
		res = merged
	}
	return res
}

type IdBuilder struct {
	Host string
}

func (idb *IdBuilder) ActivityUrl(id uint64) string {
	return fmt.Sprintf("https://%s/activity/%d", idb.Host, id)
}

func (idb *IdBuilder) SiteUrl() string {
	return fmt.Sprintf("https://%s", idb.Host)
}

func (idb *IdBuilder) SharedInbox() string {
	return fmt.Sprintf("https://%s/inbox", idb.Host)
}

func (idb *IdBuilder) UserProfile(user string) string {
	return fmt.Sprintf("https://%s/web/feeds/%s", idb.Host, user)
}

func (idb *IdBuilder) UserUrl(user string) string {
	return fmt.Sprintf("https://%s/u/%s", idb.Host, user)
}

func (idb *IdBuilder) UserKeyId(user string) string {
	return fmt.Sprintf("https://%s/u/%s#main-key", idb.Host, user)
}

func (idb *IdBuilder) UserInbox(user string) string {
	return fmt.Sprintf("https://%s/u/%s/inbox", idb.Host, user)
}

func (idb *IdBuilder) UserOutbox(user string) string {
	return fmt.Sprintf("https://%s/u/%s/outbox", idb.Host, user)
}

func (idb *IdBuilder) UserFollowing(user string) string {
	return fmt.Sprintf("https://%s/u/%s/following", idb.Host, user)
}

func (idb *IdBuilder) UserFollowers(user string) string {
	return fmt.Sprintf("https://%s/u/%s/followers", idb.Host, user)
}

func (idb *IdBuilder) UserStatus(user string, id uint64) string {
	idStr := strconv.FormatUint(id, 10)
	return fmt.Sprintf("https://%s/u/%s/status/%s", idb.Host, user, idStr)
}

func (idb *IdBuilder) UserStatusActivity(user string, id uint64) string {
	idStr := strconv.FormatUint(id, 10)
	return fmt.Sprintf("https://%s/u/%s/status/%s/activity", idb.Host, user, idStr)
}
