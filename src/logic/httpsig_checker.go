package logic

import (
	"log"
	"net/http"
	"regexp"
	"rss_parrot/dto"
)

type IHttpSigChecker interface {
	Check(w http.ResponseWriter, r *http.Request) (*dto.UserInfo, bool)
}

type httpSigChecker struct {
	userRetriever IUserRetriever
	reKeyId       *regexp.Regexp
}

func NewHttpSigChecker(userRetriever IUserRetriever) IHttpSigChecker {
	reKeyId := regexp.MustCompile("keyId=['\"]([^'\"]+)['\"]")
	return &httpSigChecker{userRetriever, reKeyId}
}

func (chk *httpSigChecker) Check(w http.ResponseWriter, r *http.Request) (*dto.UserInfo, bool) {

	var err error

	var sigHeader = r.Header.Get("Signature")
	groups := chk.reKeyId.FindStringSubmatch(sigHeader)
	if groups == nil {
		return nil, false
	}
	keyId := groups[1]

	var userInfo *dto.UserInfo
	if userInfo, err = chk.userRetriever.Retrieve(keyId); err != nil {
		log.Printf("Failed to retrieve user info for %s: %v", keyId, err)
		return nil, false
	}

	// TODO: verify signature

	return userInfo, true
}
