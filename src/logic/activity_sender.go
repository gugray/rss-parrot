package logic

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/go-fed/httpsig"
	"io"
	"net/http"
	"rss_parrot/config"
	"rss_parrot/dto"
	"strings"
	"time"
)

type IActivitySender interface {
	Send(inboxUrl string, activity dto.Activity) error
}

type ActivitySender struct {
	cfg *config.Config
}

func NewActivitySender(cfg *config.Config) IActivitySender {
	return &ActivitySender{cfg}
}

func (sender *ActivitySender) Send(inboxUrl string, activity dto.Activity) error {

	host := strings.Replace(inboxUrl, "https://", "", -1)
	slashIx := strings.IndexByte(host, '/')
	host = host[:slashIx]

	bodyJson, _ := json.Marshal(activity)
	dateStr := time.Now().UTC().Format(http.TimeFormat)

	req, err := http.NewRequest("POST", inboxUrl, bytes.NewBuffer(bodyJson))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("host", host)
	req.Header.Set("date", dateStr)

	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		[]string{httpsig.RequestTarget, "host", "date", "digest"},
		httpsig.Signature,
		0)
	if err != nil {
		return err
	}
	privkeyStr := sender.cfg.BirbPrivkey
	block, _ := pem.Decode([]byte(privkeyStr))
	privkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}
	keyId := fmt.Sprintf("https://%s/users/%s#main-key", sender.cfg.InstanceName, sender.cfg.BirbName)
	err = signer.SignRequest(privkey, keyId, req, bodyJson)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("activity POST request failed with status %s; response: %s", resp.Status, respBody)
	}

	return nil
}
