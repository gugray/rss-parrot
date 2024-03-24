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

//go:generate mockgen --build_flags=--mod=mod -destination ../test/mocks/mock_activity_sender.go -package mocks rss_parrot/logic IActivitySender

type IActivitySender interface {
	Send(privKey *rsa.PrivateKey, sendingUser, inboxUrl string, activity *dto.ActivityOut) error
}

const activityTimeoutSec = 10

type activitySender struct {
	cfg       *shared.Config
	logger    shared.ILogger
	userAgent shared.IUserAgent
	metrics   IMetrics
	idb       shared.IdBuilder
}

func NewActivitySender(cfg *shared.Config,
	logger shared.ILogger,
	userAgent shared.IUserAgent,
	metrics IMetrics,
) IActivitySender {
	return &activitySender{cfg, logger, userAgent, metrics, shared.IdBuilder{cfg.Host}}
}

func (sender *activitySender) Send(
	privKey *rsa.PrivateKey,
	sendingUser,
	inboxUrl string,
	activity *dto.ActivityOut,
) error {

	obs := sender.metrics.StartApubRequestOut("post")
	defer obs.Finish()

	host := strings.Replace(inboxUrl, "https://", "", -1)
	slashIx := strings.IndexByte(host, '/')
	if slashIx == -1 {
		return fmt.Errorf("invalid inbox url: %v", inboxUrl)
	}
	host = host[:slashIx]

	bodyJson, _ := json.Marshal(activity)
	dateStr := time.Now().UTC().Format(http.TimeFormat)

	// DBG
	//fmt.Println(string(bodyJson))

	req, err := http.NewRequest("POST", inboxUrl, bytes.NewBuffer(bodyJson))
	sender.userAgent.AddUserAgent(req)
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

	client := http.Client{}
	client.Timeout = time.Second * activityTimeoutSec
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
