package logic

import (
	"fmt"
	"net/http"
	"regexp"
	"rss_parrot/dto"
	"rss_parrot/shared"
)

type IHttpSigChecker interface {
	Check(w http.ResponseWriter, r *http.Request) (*dto.UserInfo, string)
}

type httpSigChecker struct {
	logger        shared.ILogger
	userRetriever IUserRetriever
	reKeyId       *regexp.Regexp
}

func NewHttpSigChecker(logger shared.ILogger, userRetriever IUserRetriever) IHttpSigChecker {
	reKeyId := regexp.MustCompile("keyId=['\"]([^'\"]+)['\"]")
	return &httpSigChecker{logger, userRetriever, reKeyId}
}

func (chk *httpSigChecker) Check(w http.ResponseWriter, r *http.Request) (*dto.UserInfo, string) {

	var err error

	var sigHeader = r.Header.Get("Signature")
	groups := chk.reKeyId.FindStringSubmatch(sigHeader)
	if groups == nil {
		return nil, "Missing or invalid 'Signature' header"
	}
	keyId := groups[1]

	var userInfo *dto.UserInfo
	if userInfo, err = chk.userRetriever.Retrieve(keyId); err != nil {
		chk.logger.Infof("Failed to retrieve user info for keyId %s: %v", keyId, err)
		return nil, fmt.Sprintf("Failed to retrieve user info for keyId: %s", keyId)
	}

	// TODO: verify signature

	return userInfo, ""
}
