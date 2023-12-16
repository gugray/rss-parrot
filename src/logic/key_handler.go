package logic

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"rss_parrot/dal"
	"rss_parrot/shared"
)

type IKeyHandler interface {
	GetPrivKey(user string) (*rsa.PrivateKey, error)
}

type keyHandler struct {
	cfg  *shared.Config
	repo dal.IRepo
}

func NewKeyHandler(cfg *shared.Config, repo dal.IRepo) IKeyHandler {
	return &keyHandler{cfg, repo}
}

func (kh *keyHandler) getSpecialAccountKey(user string) string {
	if user == kh.cfg.Birb.User {
		return kh.cfg.Birb.PrivKey
	}
	return ""
}

func (kh *keyHandler) GetPrivKey(user string) (*rsa.PrivateKey, error) {

	var err error

	privKeyStr := kh.getSpecialAccountKey(user)
	if privKeyStr == "" {
		privKeyStr, err = kh.repo.GetPrivKey(user)
		if err != nil {
			return nil, err
		}
	}

	block, _ := pem.Decode([]byte(privKeyStr))
	privKeyBytes := block.Bytes
	if x509.IsEncryptedPEMBlock(block) {
		privKeyBytes, err = x509.DecryptPEMBlock(block, []byte(kh.cfg.Secrets.BirdPrivKeyPass))
		if err != nil {
			return nil, err
		}
	}
	privkey, err := x509.ParsePKCS1PrivateKey(privKeyBytes)
	if err != nil {
		return nil, err
	}
	return privkey, nil
}
