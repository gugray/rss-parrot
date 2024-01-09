package logic

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/go-fed/httpsig"
	"io"
	"net/http"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"strings"
	"time"
)

type IUserRetriever interface {
	Retrieve(userUrl string) (info *dto.UserInfo, err error)
}

type userRetriever struct {
	cfg       *shared.Config
	userAgent shared.IUserAgent
	keyStore  IKeyStore
	idb       shared.IdBuilder
}

func NewUserRetriever(cfg *shared.Config, userAgent shared.IUserAgent, keyStore IKeyStore) IUserRetriever {
	return &userRetriever{cfg, userAgent, keyStore, shared.IdBuilder{cfg.Host}}
}

func (ur *userRetriever) Retrieve(userUrl string) (info *dto.UserInfo, err error) {

	host := strings.Replace(userUrl, "https://", "", -1)
	slashIx := strings.IndexByte(host, '/')
	host = host[:slashIx]
	dateStr := time.Now().UTC().Format(http.TimeFormat)

	var req *http.Request
	if req, err = http.NewRequest("GET", userUrl, nil); err != nil {
		return nil, err
	}
	ur.userAgent.AddUserAgent(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("host", host)
	req.Header.Set("date", dateStr)

	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		[]string{httpsig.RequestTarget, "Host", "date", "digest"},
		httpsig.Signature,
		0)
	if err != nil {
		return nil, err
	}

	var privKey *rsa.PrivateKey
	privKey, err = ur.keyStore.GetPrivKey(ur.cfg.Birb.User)
	if err != nil {
		return nil, err
	}
	keyId := ur.idb.UserKeyId(ur.cfg.Birb.User)
	err = signer.SignRequest(privKey, keyId, req, []byte{})
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user profile; got status %v", resp.StatusCode)
	}

	var bodyBytes []byte
	if bodyBytes, err = io.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	var obj dto.UserInfo
	if err = json.Unmarshal(bodyBytes, &obj); err != nil {
		return nil, err
	}

	return &obj, nil
}
