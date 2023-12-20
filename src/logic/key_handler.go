package logic

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"rss_parrot/dal"
	"rss_parrot/shared"
)

type IKeyHandler interface {
	GetPrivKey(user string) (*rsa.PrivateKey, error)
	MakeKeyPair() (pubKey, privKey string, err error)
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

func (kh *keyHandler) MakeKeyPair() (pubKey, privKey string, err error) {

	pubKey = ""
	privKey = ""
	err = nil

	// Generate RSA key
	var key *rsa.PrivateKey
	key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	// Extract public component.
	pub := key.Public()

	// Encode private key to PKCS#1, with password
	keyRaw := x509.MarshalPKCS1PrivateKey(key)
	encBlock, err := x509.EncryptPEMBlock(
		rand.Reader, "RSA PRIVATE KEY", keyRaw,
		[]byte(kh.cfg.Secrets.BirdPrivKeyPass), x509.PEMCipherAES256)
	if err != nil {
		return
	}
	keyPEM := pem.EncodeToMemory(encBlock)

	// Encode public key to PKCS#1
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
		},
	)

	pubKey = string(pubPEM)
	privKey = string(keyPEM)

	return
}
