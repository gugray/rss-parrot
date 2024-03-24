package logic

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"rss_parrot/dal"
	"rss_parrot/shared"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../test/mocks/mock_key_store.go -package mocks rss_parrot/logic IKeyStore

type IKeyStore interface {
	GetPrivKey(user string) (*rsa.PrivateKey, error)
	MakeKeyPair() (pubKey, privKey string, err error)
}

type keyStore struct {
	cfg  *shared.Config
	repo dal.IRepo
}

func NewKeyStore(cfg *shared.Config, repo dal.IRepo) IKeyStore {
	return &keyStore{cfg, repo}
}

func (ks *keyStore) getSpecialAccountKey(user string) string {
	if user == ks.cfg.Birb.User {
		return ks.cfg.Birb.PrivKey
	}
	return ""
}

func (ks *keyStore) GetPrivKey(user string) (*rsa.PrivateKey, error) {

	var err error

	privKeyStr := ks.getSpecialAccountKey(user)
	if privKeyStr == "" {
		privKeyStr, err = ks.repo.GetPrivKey(user)
		if err != nil {
			return nil, err
		}
	}

	block, _ := pem.Decode([]byte(privKeyStr))
	privKeyBytes := block.Bytes
	if x509.IsEncryptedPEMBlock(block) {
		privKeyBytes, err = x509.DecryptPEMBlock(block, []byte(ks.cfg.Secrets.BirdPrivKeyPass))
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

func (ks *keyStore) MakeKeyPair() (pubKey, privKey string, err error) {

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
		[]byte(ks.cfg.Secrets.BirdPrivKeyPass), x509.PEMCipherAES256)
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
