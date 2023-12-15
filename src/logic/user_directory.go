package logic

import (
	"fmt"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"strings"
	"time"
)

const pageSize = 2
const websiteLinkTemplate = "<a href='https://%s' target='_blank' rel='nofollow noopener noreferrer me' translate='no'><span class='invisible'>https://</span><span class=''>%s</span><span class='invisible'></span></a>"

type IUserDirectory interface {
	GetWebfinger(user, instance string) *dto.WebfingerResp
	GetUserInfo(user string) *dto.UserInfo
	GetOutboxSummary(user string) *dto.OrderedListSummary
	GetFollowersSummary(user string) *dto.OrderedListSummary
	GetFollowingSummary(user string) *dto.OrderedListSummary
}

type userDirectory struct {
	cfg  *shared.Config
	repo dal.IRepo
	idb  idBuilder
}

func NewUserDirectory(cfg *shared.Config, repo dal.IRepo) IUserDirectory {
	return &userDirectory{cfg, repo, idBuilder{cfg.Host}}
}

func (udir *userDirectory) GetWebfinger(user, host string) *dto.WebfingerResp {
	cfgHost := udir.cfg.Host
	cfgBirb := udir.cfg.Birb.User

	if !strings.EqualFold(host, cfgHost) || !strings.EqualFold(user, cfgBirb) {
		return nil
	}

	user = strings.ToLower(user)
	resp := dto.WebfingerResp{
		Subject: fmt.Sprintf("acct:%s@%s", user, cfgHost),
		Aliases: []string{
			udir.idb.UserProfile(user),
			udir.idb.UserUrl(user),
		},
		Links: []dto.WebfingerLink{
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: udir.idb.UserProfile(user),
			},
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: udir.idb.UserUrl(user),
			},
		},
	}
	return &resp
}

func (udir *userDirectory) GetUserInfo(user string) *dto.UserInfo {

	userInfo := udir.cfg.Birb
	if !strings.EqualFold(user, userInfo.User) {
		return nil
	}

	user = strings.ToLower(user)
	userUrl := udir.idb.UserUrl(user)

	resp := dto.UserInfo{
		Context: []string{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		Id:                userUrl,
		Type:              "Person",
		PreferredUserName: user,
		Name:              userInfo.Name,
		Summary:           userInfo.Summary,
		ManuallyApproves:  false,
		Published:         userInfo.Published.Format(time.RFC3339),
		Inbox:             udir.idb.UserInbox(user),
		Outbox:            udir.idb.UserOutbox(user),
		Followers:         udir.idb.UserFollowers(user),
		Following:         udir.idb.UserFollowing(user),
		Endpoints:         dto.UserEndpoints{SharedInbox: udir.idb.SharedInbox()},
		PublicKey: dto.PublicKey{
			Id:           udir.idb.UserKeyId(user),
			Owner:        userUrl,
			PublicKeyPem: userInfo.PubKey,
		},
		Attachments: []dto.Attachment{
			{
				Type:  "PropertyValue",
				Name:  "Website",
				Value: fmt.Sprintf(websiteLinkTemplate, udir.cfg.Host, udir.cfg.Host),
			},
		},
		Icon: dto.Image{
			Type: "Image",
			Url:  userInfo.ProfilePic,
		},
		Image: dto.Image{
			Type: "Image",
			Url:  userInfo.HeaderPic,
		},
	}
	return &resp
}

func (udir *userDirectory) GetOutboxSummary(user string) *dto.OrderedListSummary {

	cfgBirb := udir.cfg.Birb.User
	if !strings.EqualFold(user, cfgBirb) {
		return nil
	}

	user = strings.ToLower(user)

	resp := dto.OrderedListSummary{
		Context:    "https://www.w3.org/ns/activitystreams",
		Id:         udir.idb.UserUrl(user),
		Type:       "OrderedCollection",
		TotalItems: udir.repo.GetPostCount(),
	}
	return &resp
}

func (udir *userDirectory) GetFollowersSummary(user string) *dto.OrderedListSummary {

	cfgBirb := udir.cfg.Birb.User
	if !strings.EqualFold(user, cfgBirb) {
		return nil
	}

	user = strings.ToLower(user)

	resp := dto.OrderedListSummary{
		Context:    "https://www.w3.org/ns/activitystreams",
		Id:         udir.idb.UserFollowers(user),
		Type:       "OrderedCollection",
		TotalItems: udir.repo.GetPostCount(),
	}
	return &resp
}

func (udir *userDirectory) GetFollowingSummary(user string) *dto.OrderedListSummary {

	cfgBirb := udir.cfg.Birb.User
	if !strings.EqualFold(user, cfgBirb) {
		return nil
	}

	user = strings.ToLower(user)

	resp := dto.OrderedListSummary{
		Context:    "https://www.w3.org/ns/activitystreams",
		Id:         udir.idb.UserFollowers(user),
		Type:       "OrderedCollection",
		TotalItems: udir.repo.GetPostCount(),
	}
	return &resp
}
