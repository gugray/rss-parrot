package logic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"time"
)

type IInbox interface {
	HandleFollow(receivingUser string, senderInfo *dto.UserInfo, bodyBytes []byte) (string, error)
	HandleUndo(receivingUser string, senderInfo *dto.UserInfo, bodyBytes []byte) (string, error)
}

type inbox struct {
	cfg             *shared.Config
	logger          shared.ILogger
	repo            dal.IRepo
	sender          IActivitySender
	reUserUrlParser *regexp.Regexp
}

func NewInbox(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	sender IActivitySender,
) IInbox {
	reUserUrlParser := regexp.MustCompile("https://" + cfg.Host + "/users/([^/]+)/?")
	return &inbox{cfg, logger, repo, sender, reUserUrlParser}
}

func (ib *inbox) HandleFollow(
	receivingUser string,
	senderInfo *dto.UserInfo,
	bodyBytes []byte) (reqProblem string, err error) {

	ib.logger.Infof("Handling Follow activity to %s", receivingUser)

	reqProblem = ""
	err = nil

	if receivingUser != ib.cfg.BirbName {
		reqProblem = fmt.Sprintf("User does not exist: %s", receivingUser)
		return
	}

	var actFollow dto.ActivityIn[string]
	if jsonErr := json.Unmarshal(bodyBytes, &actFollow); jsonErr != nil {
		ib.logger.Info("Invalid JSON in Follow activity body")
		reqProblem = fmt.Sprintf("Invalid JSON: %d", jsonErr)
		return
	}

	// Is object the ID if this account?
	cfgInstance := ib.cfg.Host
	myUserId := fmt.Sprintf("https://%s/users/%s", cfgInstance, receivingUser)
	if myUserId != actFollow.Object {
		msg := fmt.Sprintf("Follow activity sent to inbox of %s, but object is %s", receivingUser, actFollow.Object)
		ib.logger.Warn(msg)
		reqProblem = msg
		return
	}

	// Store new follower
	var actorUrl *url.URL
	var urlError error
	actorUrl, urlError = url.Parse(actFollow.Actor)
	if urlError != nil {
		msg := fmt.Sprintf("Failed to parse actor URL '%s': %v", actFollow.Actor, urlError)
		ib.logger.Warn(msg)
		reqProblem = msg
		return
	}

	ib.repo.AddFollower(&dal.Follower{
		User:        actFollow.Actor,
		Handle:      senderInfo.PreferredUserName,
		Host:        actorUrl.Hostname(),
		SharedInbox: senderInfo.Endpoints.SharedInbox,
	})

	go ib.sendFollowAccept(receivingUser, senderInfo.Inbox, &actFollow)

	return
}

func (ib *inbox) sendFollowAccept(followedUserName, inboxUrl string, actFollow *dto.ActivityIn[string]) {

	time.Sleep(3000)

	ib.logger.Infof("Sending 'Accept' to %s", inboxUrl)

	actAccept := dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      actFollow.Object,
		Type:    "Accept",
		Actor:   actFollow.Object,
		Object:  actFollow,
	}

	if err := ib.sender.Send(followedUserName, inboxUrl, &actAccept); err != nil {
		ib.logger.Infof("Failed to send 'Accept' activity: %v", err)
	}
}

func (ib *inbox) HandleUndo(
	receivingUser string,
	senderInfo *dto.UserInfo,
	bodyBytes []byte) (reqProblem string, err error) {

	ib.logger.Infof("Handling Undo activity to %s", receivingUser)

	reqProblem = ""
	err = nil

	if receivingUser != ib.cfg.BirbName {
		reqProblem = fmt.Sprintf("User does not exist: %s", receivingUser)
		return
	}

	var actUndo dto.ActivityIn[dto.ActivityInBase]
	if jsonErr := json.Unmarshal(bodyBytes, &actUndo); jsonErr != nil {
		ib.logger.Info("Invalid JSON in Undo activity body")
		reqProblem = fmt.Sprintf("Invalid JSON: %d", jsonErr)
		return
	}

	// Undoing what?
	if actUndo.Object.Type == "Follow" {
		reqProblem, err = ib.handleUnfollow(receivingUser, bodyBytes)
	}

	return
}

func (ib *inbox) handleUnfollow(receivingUser string, bodyBytes []byte) (reqProblem string, err error) {

	ib.logger.Infof("Handling Undo Follow activity to %s", receivingUser)

	reqProblem = ""
	err = nil

	// Now parse the embeded object
	var actUndoFollow dto.ActivityIn[dto.ActivityIn[string]]
	if jsonErr := json.Unmarshal(bodyBytes, &actUndoFollow); jsonErr != nil {
		ib.logger.Info("Invalid JSON in Undo Follow activity body")
		reqProblem = fmt.Sprintf("Invalid JSON: %d", jsonErr)
		return
	}

	// Who is being unfollowed, according to the object?
	groups := ib.reUserUrlParser.FindStringSubmatch(actUndoFollow.Object.Object)
	if groups == nil {
		reqProblem = fmt.Sprintf("Cannot parse Undo Follow object as a URL: %s", actUndoFollow.Object.Object)
		return
	}
	objectUser := groups[1]
	if objectUser != receivingUser {
		reqProblem = fmt.Sprintf("Undo Follow sent to '%s' but user in object URL us %s", receivingUser, objectUser)
		return
	}

	ib.repo.RemoveFollower(actUndoFollow.Actor)

	return
}
