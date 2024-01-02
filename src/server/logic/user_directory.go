package logic

import (
	"fmt"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"rss_parrot/texts"
	"strings"
	"time"
)

const pageSize = 2
const websiteLinkTemplate = "<a href='%s' target='_blank' rel='nofollow noopener noreferrer me' translate='no'>%s</a>"

// TODO: return error in all of these

type IUserDirectory interface {
	GetWebfinger(user string) *dto.WebfingerResp
	GetUserInfo(user string) *dto.UserInfo
	GetOutboxSummary(user string) *dto.OrderedListSummary
	GetFollowersSummary(user string) *dto.OrderedListSummary
	GetFollowingSummary(user string) *dto.OrderedListSummary
	AcceptFollower(followActId, followerUserUrl, followerInbox, followedUser string) error
}

type userDirectory struct {
	cfg      *shared.Config
	logger   shared.ILogger
	repo     dal.IRepo
	idb      shared.IdBuilder
	keyStore IKeyStore
	sender   IActivitySender
	txt      texts.ITexts
}

func NewUserDirectory(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	keyStore IKeyStore,
	sender IActivitySender,
	txt texts.ITexts,
) IUserDirectory {
	return &userDirectory{
		cfg:      cfg,
		logger:   logger,
		repo:     repo,
		idb:      shared.IdBuilder{cfg.Host},
		keyStore: keyStore,
		sender:   sender,
		txt:      txt}
}

func (udir *userDirectory) GetWebfinger(user string) *dto.WebfingerResp {

	cfgHost := udir.cfg.Host
	acct, err := udir.repo.GetAccount(user)
	if err != nil || acct == nil {
		return nil // TODO errors
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
func (udir *userDirectory) getWebsiteAttachment(url string) string {
	justUrl := strings.TrimPrefix(url, "https://")
	justUrl = strings.TrimPrefix(url, "http://")
	return fmt.Sprintf(websiteLinkTemplate, url, justUrl)
}

func (udir *userDirectory) fillBirbUserInfo(ui *dto.UserInfo) {
	ui.Name = udir.txt.Get("birb_name.txt")
	ui.Summary = udir.txt.Get("birb_bio.html")
	ui.ManuallyApproves = udir.cfg.Birb.ManuallyApprovesFollows
	ui.PublicKey = dto.PublicKey{
		Id:           udir.idb.UserKeyId(udir.cfg.Birb.User),
		Owner:        ui.Id,
		PublicKeyPem: udir.cfg.Birb.PubKey,
	}
	ui.Attachments = append(ui.Attachments, dto.Attachment{
		Type:  "PropertyValue",
		Name:  "Website",
		Value: udir.getWebsiteAttachment(udir.idb.SiteUrl()),
	})
	ui.Icon = dto.Image{
		Type: "Image",
		Url:  udir.cfg.Birb.ProfilePic,
	}
	ui.Image = dto.Image{
		Type: "Image",
		Url:  udir.cfg.Birb.HeaderPic,
	}
}

func (udir *userDirectory) fillFeedUserInfo(ui *dto.UserInfo, acct *dal.Account) {
	ui.Name = shared.GetNameWithParrot(acct.FeedName)
	ui.Summary = udir.txt.WithVals("acct_bio.html", map[string]string{
		"siteUrl":     udir.idb.SiteUrl(),
		"description": acct.FeedSummary,
	})
	ui.ManuallyApproves = false
	ui.PublicKey = dto.PublicKey{
		Id:           udir.idb.UserKeyId(acct.Handle),
		Owner:        ui.Id,
		PublicKeyPem: acct.PubKey,
	}
	ui.Attachments = append(ui.Attachments, dto.Attachment{
		Type:  "PropertyValue",
		Name:  "Website",
		Value: udir.getWebsiteAttachment(acct.SiteUrl),
	})
	ui.Icon = dto.Image{
		Type: "Image",
		Url:  acct.ProfileImageUrl,
	}
	ui.Image = dto.Image{
		Type: "Image",
		Url:  acct.HeaderImageUrl,
	}
	if ui.Icon.Url == "" {
		ui.Icon.Url = udir.cfg.FallbackProfilePic
	}
}

func (udir *userDirectory) GetUserInfo(user string) *dto.UserInfo {

	user = strings.ToLower(user)
	userUrl := udir.idb.UserUrl(user)
	acct, err := udir.repo.GetAccount(user)
	if err != nil || acct == nil {
		return nil // TODO errors
	}

	resp := dto.UserInfo{
		Context: []string{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		Id:                userUrl,
		Type:              "Service",
		PreferredUserName: user,
		Published:         acct.CreatedAt.Format(time.RFC3339),
		Inbox:             udir.idb.UserInbox(user),
		Outbox:            udir.idb.UserOutbox(user),
		Followers:         udir.idb.UserFollowers(user),
		Following:         udir.idb.UserFollowing(user),
		Endpoints:         dto.UserEndpoints{SharedInbox: udir.idb.SharedInbox()},
		Attachments:       []dto.Attachment{},
	}

	if user == udir.cfg.Birb.User {
		udir.fillBirbUserInfo(&resp)
	} else {
		udir.fillFeedUserInfo(&resp, acct)
	}

	return &resp
}

func (udir *userDirectory) GetOutboxSummary(user string) *dto.OrderedListSummary {

	var err error
	var exists bool
	user = strings.ToLower(user)
	exists, err = udir.repo.DoesAccountExist(user)
	if err != nil || !exists {
		return nil // TODO errors
	}

	var postCount uint
	postCount, err = udir.repo.GetPostCount(user) // TODO errors

	resp := dto.OrderedListSummary{
		Context:    "https://www.w3.org/ns/activitystreams",
		Id:         udir.idb.UserUrl(user),
		Type:       "OrderedCollection",
		TotalItems: postCount,
	}
	return &resp
}

func (udir *userDirectory) GetFollowersSummary(user string) *dto.OrderedListSummary {

	var err error
	var exists bool
	user = strings.ToLower(user)
	exists, err = udir.repo.DoesAccountExist(user)
	if err != nil || !exists {
		return nil // TODO errors
	}

	var followerCount uint
	followerCount, err = udir.repo.GetApprovedFollowerCount(user) // TODO errors

	resp := dto.OrderedListSummary{
		Context:    "https://www.w3.org/ns/activitystreams",
		Id:         udir.idb.UserFollowers(user),
		Type:       "OrderedCollection",
		TotalItems: followerCount,
	}
	return &resp
}

func (udir *userDirectory) GetFollowingSummary(user string) *dto.OrderedListSummary {

	var err error
	var exists bool
	user = strings.ToLower(user)
	exists, err = udir.repo.DoesAccountExist(user)
	if err != nil || !exists {
		return nil // TODO errors
	}

	resp := dto.OrderedListSummary{
		Context:    "https://www.w3.org/ns/activitystreams",
		Id:         udir.idb.UserFollowers(user),
		Type:       "OrderedCollection",
		TotalItems: 0,
	}
	return &resp
}

func (udir *userDirectory) AcceptFollower(followActId, followerUserUrl, followerInbox, followedUser string) error {

	udir.logger.Infof("Accepting follow %s", followerInbox)

	privKey, err := udir.keyStore.GetPrivKey(followedUser)
	if err != nil {
		err = fmt.Errorf("failed to get private key for user %s: %v", followedUser, err)
		return err
	}

	acceptId := udir.repo.GetNextId()

	actAccept := dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      udir.idb.ActivityUrl(acceptId),
		Type:    "Accept",
		Actor:   udir.idb.UserUrl(followedUser),
		Object: dto.ActivityOut{
			Id:     followActId,
			Type:   "Follow",
			Actor:  followerUserUrl,
			Object: udir.idb.UserUrl(followedUser),
		},
	}

	if err = udir.sender.Send(privKey, followedUser, followerInbox, &actAccept); err != nil {
		err = fmt.Errorf("failed to send 'Accept' activity: %v", err)
		return err
	}

	if err = udir.repo.SetFollowerApproveStatus(followedUser, followerUserUrl, 1); err != nil {
		err = fmt.Errorf("failed set follower approve status: %v", err)
		return err
	}

	return nil
}
