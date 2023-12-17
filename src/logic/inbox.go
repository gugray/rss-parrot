package logic

import (
	"encoding/json"
	"fmt"
	"regexp"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"rss_parrot/texts"
	"time"
)

type IInbox interface {
	HandleFollow(receivingUser string, senderInfo *dto.UserInfo, bodyBytes []byte) (string, error)
	HandleUndo(receivingUser string, senderInfo *dto.UserInfo, bodyBytes []byte) (string, error)
	HandleCreateNote(actBase dto.ActivityInBase, senderInfo *dto.UserInfo, bodyBytes []byte) (string, error)
}

type inbox struct {
	cfg             *shared.Config
	logger          shared.ILogger
	idb             shared.IdBuilder
	repo            dal.IRepo
	txt             texts.ITexts
	keyHandler      IKeyHandler
	sender          IActivitySender
	messenger       IMessenger
	reUserUrlParser *regexp.Regexp
}

func NewInbox(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	txt texts.ITexts,
	keyHandler IKeyHandler,
	sender IActivitySender,
	messenger IMessenger,
) IInbox {
	reUserUrlParser := regexp.MustCompile("https://" + cfg.Host + "/u/([^/]+)/?")
	return &inbox{cfg, logger, shared.IdBuilder{cfg.Host}, repo, txt, keyHandler,
		sender, messenger, reUserUrlParser}
}

func (ib *inbox) HandleFollow(
	receivingUser string,
	senderInfo *dto.UserInfo,
	bodyBytes []byte) (reqProblem string, err error) {

	ib.logger.Infof("Handling Follow activity to '%s'", receivingUser)

	reqProblem = ""
	err = nil
	var account *dal.Account
	account, err = ib.repo.GetAccount(receivingUser)
	if err != nil {
		return "", err
	}
	if account == nil {
		reqProblem = fmt.Sprintf("User does not exist: %s", receivingUser)
		return
	}

	// Is this a built-in account (ie not a feed parrot)?
	// Those are not taking followers
	if account.RssUrl == "" {
		return "", nil
	}

	var actFollow dto.ActivityIn[string]
	if jsonErr := json.Unmarshal(bodyBytes, &actFollow); jsonErr != nil {
		ib.logger.Info("Invalid JSON in Follow activity body")
		reqProblem = fmt.Sprintf("Invalid JSON: %d", jsonErr)
		return
	}

	// Is object the ID if this account?
	myUserUrl := ib.idb.UserUrl(receivingUser)
	if myUserUrl != actFollow.Object {
		msg := fmt.Sprintf("Follow activity sent to inbox of %s, but object is %s", receivingUser, actFollow.Object)
		ib.logger.Warn(msg)
		reqProblem = msg
		return
	}

	// Store new follower
	var actorHostName string
	var urlError error
	actorHostName, urlError = shared.GetHostName(actFollow.Actor)
	if urlError != nil {
		ib.logger.Warn(urlError.Error())
		reqProblem = urlError.Error()
		return
	}

	err = ib.repo.AddFollower(receivingUser, &dal.MastodonUserInfo{
		UserUrl:     actFollow.Actor,
		Handle:      senderInfo.PreferredUserName,
		Host:        actorHostName,
		SharedInbox: senderInfo.Endpoints.SharedInbox,
	})
	if err != nil {
		return "", err
	}

	go ib.sendFollowAccept(receivingUser, senderInfo.Inbox, &actFollow)

	return
}

func (ib *inbox) sendFollowAccept(followedUserName, inboxUrl string, actFollow *dto.ActivityIn[string]) {

	time.Sleep(1000)

	ib.logger.Infof("Sending 'Accept' to %s", inboxUrl)

	privKey, err := ib.keyHandler.GetPrivKey(followedUserName)
	if err != nil {
		ib.logger.Errorf("Failed to private key for user %s: %v", followedUserName, err)
		return
	}

	actAccept := dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      actFollow.Object,
		Type:    "Accept",
		Actor:   actFollow.Object,
		Object:  actFollow,
	}

	if err = ib.sender.Send(privKey, followedUserName, inboxUrl, &actAccept); err != nil {
		ib.logger.Warnf("Failed to send 'Accept' activity: %v", err)
	}
}

func (ib *inbox) HandleUndo(
	receivingUser string,
	senderInfo *dto.UserInfo,
	bodyBytes []byte) (reqProblem string, err error) {

	ib.logger.Infof("Handling Undo activity to %s", receivingUser)

	reqProblem = ""
	err = nil

	if receivingUser != ib.cfg.Birb.User {
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

	ib.logger.Infof("Handling Undo Follow activity to '%s'", receivingUser)

	reqProblem = ""
	err = nil
	var userExists bool
	userExists, err = ib.repo.DoesAccountExist(receivingUser)
	if err != nil {
		return
	}
	if !userExists {
		reqProblem = fmt.Sprintf("User does not exist: %s", receivingUser)
		return
	}

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

	err = ib.repo.RemoveFollower(receivingUser, actUndoFollow.Actor)

	return
}

func (ib *inbox) HandleCreateNote(
	actBase dto.ActivityInBase,
	senderInfo *dto.UserInfo,
	bodyBytes []byte) (reqProblem string, err error) {

	ib.logger.Infof("Handling Create Note activity")

	reqProblem = ""
	err = nil

	// Is it addressed to both me, and "public"?
	birbUsrUrl := ib.idb.UserUrl(ib.cfg.Birb.User)
	toMe := false
	toPublicOrFollowers := false
	checkAddressee := func(str string) {
		if str == shared.ActivityPublic || str == senderInfo.Followers {
			toPublicOrFollowers = true
		} else if str == birbUsrUrl {
			toMe = true
		}
	}
	for _, str := range actBase.To {
		checkAddressee(str)
	}
	for _, str := range actBase.Cc {
		checkAddressee(str)
	}

	// If not addressed to me: simply ignore
	if !toMe {
		return
	}

	// Parse activity with Note object
	var act dto.ActivityIn[dto.Note]
	if jsonErr := json.Unmarshal(bodyBytes, &act); jsonErr != nil {
		ib.logger.Info("Invalid JSON in Create Note activity body")
		reqProblem = fmt.Sprintf("Invalid JSON: %d", jsonErr)
		return
	}

	// So, we will reply *something* with a mention.
	// Let's get the sender's moniker! -> @twilliability@genart.social
	var senderHostName string
	senderHostName, err = shared.GetHostName(senderInfo.Id)
	if err != nil {
		reqProblem = fmt.Sprintf("Failed to extract host from sender ID %s: %d", senderInfo.Id, err)
		return
	}
	moniker := shared.MakeFullMoniker(senderHostName, senderInfo.PreferredUserName)

	// If not public: reply that we don't do DMs
	if !toPublicOrFollowers {
		msg := ib.txt.WithVals("reply_no_dm.html", map[string]string{
			"moniker": moniker,
			"userUrl": senderInfo.Id,
		})
		go ib.messenger.SendReply(ib.cfg.Birb.User, moniker, actBase.Actor, senderInfo.Inbox, act.Object.Id, msg)
		return
	}

	// TODO
	// Parse URL out of message
	// We need a 'messenger' that can send immediate and scheduled message. Swallows 'broadcaster'.
	// We need a 'feed_follower' that can retrieve initial feed info, and updates
	// We need a 'scheduled_poller' that automates/coordinates feed_follower and broadcaster
	// Go -> [init feed / find existing; send reply based on outcome]

	return
}
