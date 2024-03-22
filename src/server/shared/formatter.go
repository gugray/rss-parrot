package shared

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

const MaxDescriptionLen = 256

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
	if len(text) <= maxLen {
		return text
	}
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

func ValidateHandle(handle string) error {
	if len(handle) == 0 {
		return errors.New("parrot handle cannot be empty")
	}
	var nDots, nNonDots, nUpper int
	for _, c := range handle {
		if unicode.IsUpper(c) {
			nUpper++
		}
		if c == '.' {
			nDots++
		} else {
			nNonDots++
		}
	}
	if nDots == 0 {
		return errors.New("parrot handle must have at least one dot")
	}
	if nNonDots < 2 {
		return errors.New("parrot handle must have at least two non-dots")
	}
	if nUpper != 0 {
		return errors.New("parrot handle must not have upper-case letters")
	}
	return nil
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
