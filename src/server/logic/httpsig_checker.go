package logic

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/go-fed/httpsig"
	"net/http"
	"regexp"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"strings"
)

type IHttpSigChecker interface {
	Check(actor string, w http.ResponseWriter, r *http.Request) (*dto.UserInfo, string, error)
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

func (chk *httpSigChecker) Check(actor string, w http.ResponseWriter, r *http.Request) (*dto.UserInfo, string, error) {

	var err error

	var sigHeader = r.Header.Get("Signature")
	groups := chk.reKeyId.FindStringSubmatch(sigHeader)
	if groups == nil {
		return nil, "Missing or invalid 'Signature' header", nil
	}
	keyId := groups[1]

	if !strings.HasPrefix(keyId, actor) {
		return nil, fmt.Sprintf("Actor is not prefix of keyId; actor: %s, keyId: %s", actor, keyId), nil
	}

	var userInfo *dto.UserInfo
	if userInfo, err = chk.userRetriever.Retrieve(actor); err != nil {
		return nil, fmt.Sprintf("Failed to retrieve user info for actor: %s: %v", actor, err), nil
	}

	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		chk.logger.Errorf("Failed to create signature verifier: %v", err)
		return nil, "", err
	}

	pubKeyStr := userInfo.PublicKey.PublicKeyPem
	block, _ := pem.Decode([]byte(pubKeyStr))

	var pubKey interface{}
	if pubKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		return nil, fmt.Sprintf("Failed to parse sender's public key: %v", err), nil
	}

	if err = verifier.Verify(pubKey, httpsig.RSA_SHA256); err != nil {
		return nil, fmt.Sprintf("Incorrect signature: %v", err), nil
	}

	return userInfo, "", nil
}
