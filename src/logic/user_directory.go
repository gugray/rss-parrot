package logic

import (
	"fmt"
	"rss_parrot/dto"
	"strings"
)

type UserDirectory struct {
	cfg *Config
}

func NewUserDirectory(cfg *Config) *UserDirectory {
	return &UserDirectory{cfg}
}

func (udir *UserDirectory) GetUserInfo(user string) *dto.UserInfo {

	cfgInstance := udir.cfg.InstanceName
	cfgBirb := udir.cfg.BirbName
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
			PublicKeyPem: udir.cfg.BirbPubkey,
		},
	}
	return &resp
}
