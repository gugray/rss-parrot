package logic

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-fed/httpsig"
	"io"
	"net/http"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"strings"
	"time"
)

type IActivitySender interface {
	Send(privKey *rsa.PrivateKey, sendingUser, inboxUrl string, activity *dto.ActivityOut) error
}

type activitySender struct {
	cfg    *shared.Config
	logger shared.ILogger
	idb    shared.IdBuilder
}

func NewActivitySender(cfg *shared.Config, logger shared.ILogger) IActivitySender {
	return &activitySender{cfg, logger, shared.IdBuilder{cfg.Host}}
}

func (sender *activitySender) Send(
	privKey *rsa.PrivateKey,
	sendingUser,
	inboxUrl string,
	activity *dto.ActivityOut,
) error {

	host := strings.Replace(inboxUrl, "https://", "", -1)
	slashIx := strings.IndexByte(host, '/')
	host = host[:slashIx]

	bodyJson, _ := json.Marshal(activity)
	dateStr := time.Now().UTC().Format(http.TimeFormat)

	// DBG
	//fmt.Println(string(bodyJson))

	req, err := http.NewRequest("POST", inboxUrl, bytes.NewBuffer(bodyJson))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("host", host)
	req.Header.Set("date", dateStr)

	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		[]string{httpsig.RequestTarget, "Host", "date", "digest"},
		httpsig.Signature,
		0)
	if err != nil {
		return err
	}

	keyId := sender.idb.UserKeyId(sendingUser)
	err = signer.SignRequest(privKey, keyId, req, bodyJson)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	// DBG
	//sender.logger.Debug(string(respBody))

	if resp.StatusCode >= 300 {
		msg := fmt.Sprintf("got status %s: response: %s", resp.Status, respBody)
		sender.logger.Warnf("Activity POST failed: %s", msg)
		return errors.New(msg)
	}

	return nil
}
