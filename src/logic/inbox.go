package logic

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"time"
)

type IInbox interface {
	HandleFollow(followedUserName string, senderInfo *dto.UserInfo, bodyBytes []byte) (badReq, err error)
}

type inbox struct {
	cfg    *shared.Config
	repo   dal.IRepo
	sender IActivitySender
}

func NewInbox(
	cfg *shared.Config,
	repo dal.IRepo,
	sender IActivitySender,
) IInbox {
	return &inbox{cfg, repo, sender}
}

func (ib *inbox) HandleFollow(followedUserName string, senderInfo *dto.UserInfo, bodyBytes []byte) (badReq, err error) {

	badReq = nil
	err = nil

	if followedUserName != ib.cfg.BirbName {
		badReq = fmt.Errorf("user does not exist: %s", followedUserName)
		return
	}

	var actFollow dto.ActivityInStringObject
	if jsonErr := json.Unmarshal(bodyBytes, &actFollow); jsonErr != nil {
		log.Printf("Invalid JSON in Follow request body")
		badReq = jsonErr
		return
	}

	// Is object the ID if this account?
	cfgInstance := ib.cfg.InstanceName
	cfgBirb := ib.cfg.BirbName
	myUserId := fmt.Sprintf("https://%s/users/%s", cfgInstance, cfgBirb)
	if myUserId != actFollow.Object {
		log.Printf("Follow request sent to inbox of %s, but object is %s", followedUserName, actFollow.Object)
		badReq = fmt.Errorf("wrong inbox")
		return
	}

	// Store new follower
	var actorUrl *url.URL
	actorUrl, badReq = url.Parse(actFollow.Actor)
	if badReq != nil {
		return
	}

	ib.repo.AddFollower(&dal.Follower{
		User:        actFollow.Actor,
		Handle:      senderInfo.PreferredUserName,
		Host:        actorUrl.Hostname(),
		SharedInbox: senderInfo.Endpoints.SharedInbox,
	})

	go ib.sendFollowAccept(senderInfo.Inbox, &actFollow)

	return
}

func (ib *inbox) sendFollowAccept(inboxUrl string, actFollow *dto.ActivityInStringObject) {

	time.Sleep(3000)

	log.Printf("Sending 'Accept' to %s", inboxUrl)

	actAccept := dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      actFollow.Object,
		Type:    "Accept",
		Actor:   actFollow.Object,
		Object:  actFollow,
	}

	if err := ib.sender.Send(inboxUrl, &actAccept); err != nil {
		log.Printf("Failed to send 'Accept' activity: %v", err)
	}
}
