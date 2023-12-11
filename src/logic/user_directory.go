package logic

import (
	"fmt"
	"rss_parrot/config"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"strings"
)

type IUserDirectory interface {
	GetUserInfo(user string) *dto.UserInfo
}

type UserDirectory struct {
	cfg  *config.Config
	repo dal.IRepo
}

func NewUserDirectory(cfg *config.Config, repo dal.IRepo) IUserDirectory {
	return &UserDirectory{cfg, repo}
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
		Outbox:            fmt.Sprintf("%s/outbox", userId),
		PublicKey: dto.PublicKey{
			Id:           fmt.Sprintf("%s#main-key", userId),
			Owner:        userId,
			PublicKeyPem: udir.cfg.BirbPubkey,
		},
	}
	return &resp
}
