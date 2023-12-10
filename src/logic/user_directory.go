package logic

import (
	"fmt"
	"rss_parrot/dto"
	"strings"
)

type UserDirectoryConfig interface {
	GetInstanceName() string
	GetBirbName() string
	GetBirbPubkey() string
}

type UserDirectory struct {
	cfg UserDirectoryConfig
}

func NewUserDirectory(cfg UserDirectoryConfig) *UserDirectory {
	return &UserDirectory{cfg}
}

func (udir *UserDirectory) GetUserInfo(user string) *dto.UserInfo {

	cfgInstance := udir.cfg.GetInstanceName()
	cfgBirb := udir.cfg.GetBirbName()
	if !strings.EqualFold(user, cfgBirb) {
		return nil
	}

	user = strings.ToLower(user)
	userId := fmt.Sprintf("https://%s/users/%s", cfgInstance, user)

	resp := dto.UserInfo{
		Context: []string{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		Id:                userId,
		Type:              "Person",
		PreferredUserName: user,
		Inbox:             fmt.Sprintf("%s/inbox", userId),
		PublicKey: dto.PublicKey{
			Id:           fmt.Sprintf("%s#main-key", userId),
			Owner:        userId,
			PublicKeyPem: udir.cfg.GetBirbPubkey(),
		},
	}
	return &resp
}
