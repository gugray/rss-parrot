package logic

import (
	"fmt"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"strings"
)

type IUserDirectory interface {
	GetUserInfo(user string) *dto.UserInfo
}

type userDirectory struct {
	cfg  *shared.Config
	repo dal.IRepo
}

func NewUserDirectory(cfg *shared.Config, repo dal.IRepo) IUserDirectory {
	return &userDirectory{cfg, repo}
}

func (udir *userDirectory) GetUserInfo(user string) *dto.UserInfo {

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
		Name:              "Birby Mc Birb",
		Summary:           "Psittaciform diversity in South America and Australasia suggests that the order may have evolved in Gondwana, centred in Australasia.",
		ManuallyApproves:  false,
		Published:         "2018-04-23T22:05:35Z",
		Inbox:             fmt.Sprintf("%s/inbox", userId),
		Outbox:            fmt.Sprintf("%s/outbox", userId),
		Followers:         fmt.Sprintf("%s/followers", userId),
		Following:         fmt.Sprintf("%s/following", userId),
		Endpoints:         dto.UserEndpoints{SharedInbox: fmt.Sprintf("https://%s/inbox", cfgInstance)},
		PublicKey: dto.PublicKey{
			Id:           fmt.Sprintf("%s#main-key", userId),
			Owner:        userId,
			PublicKeyPem: udir.cfg.BirbPubkey,
		},
	}
	return &resp
}
