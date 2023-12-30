package logic

import (
	"encoding/json"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"net/url"
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
	udir            IUserDirectory
	keyStore        IKeyStore
	sender          IActivitySender
	messenger       IMessenger
	fdfol           IFeedFollower
	reUserUrlParser *regexp.Regexp
	reHttps         *regexp.Regexp
}

func NewInbox(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	txt texts.ITexts,
	udir IUserDirectory,
	keyStore IKeyStore,
	sender IActivitySender,
	messenger IMessenger,
	fdfol IFeedFollower,
) IInbox {
	reUserUrlParser := regexp.MustCompile("https://" + cfg.Host + "/u/([^/]+)/?")
	reHttps := regexp.MustCompile("https?://[^ ]+")
	return &inbox{cfg, logger, shared.IdBuilder{cfg.Host}, repo, txt, udir,
		keyStore, sender, messenger, fdfol,
		reUserUrlParser, reHttps}
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

	// Unmarshal as Follow activity
	var actFollow dto.ActivityIn[string]
	if jsonErr := json.Unmarshal(bodyBytes, &actFollow); jsonErr != nil {
		ib.logger.Info("Invalid JSON in Follow activity body")
		reqProblem = fmt.Sprintf("Invalid JSON: %d", jsonErr)
		return
	}

	// This activity already handled?
	var alreadyHandled bool
	alreadyHandled, err = ib.repo.MarkActivityHandled(actFollow.Id, time.Now())
	if err != nil {
		return
	}
	if alreadyHandled {
		ib.logger.Infof("Activity has already been handled: %s", actFollow.Id)
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

	flwr := dal.FollowerInfo{
		RequestId:     actFollow.Id,
		ApproveStatus: 0,
		UserUrl:       actFollow.Actor,
		Handle:        senderInfo.PreferredUserName,
		Host:          actorHostName,
		UserInbox:     senderInfo.Inbox,
		SharedInbox:   senderInfo.Endpoints.SharedInbox,
	}
	err = ib.repo.AddFollower(receivingUser, &flwr)
	if err != nil {
		return "", err
	}

	autoAccept := true
	if account.Handle == ib.cfg.Birb.User && ib.cfg.Birb.ManuallyApprovesFollows {
		autoAccept = false
	}
	if autoAccept {
		go func() {
			time.Sleep(1000)
			err := ib.udir.AcceptFollower(flwr.RequestId, flwr.UserUrl, flwr.UserInbox, receivingUser)
			if err != nil {
				ib.logger.Errorf("Error accepting follower: %v", err)
			}
		}()
	}

	return
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

	// This activity already handled?
	var alreadyHandled bool
	alreadyHandled, err = ib.repo.MarkActivityHandled(actUndo.Id, time.Now())
	if err != nil {
		return
	}
	if alreadyHandled {
		ib.logger.Infof("Activity has already been handled: %s", actUndo.Id)
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

	// This activity already handled?
	var alreadyHandled bool
	alreadyHandled, err = ib.repo.MarkActivityHandled(actBase.Id, time.Now())
	if err != nil {
		return
	}
	if alreadyHandled {
		ib.logger.Infof("Activity has already been handled: %s", actBase.Id)
		return
	}

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
		ib.logger.Info(reqProblem)
		return
	}
	moniker := shared.MakeFullMoniker(senderHostName, senderInfo.PreferredUserName)

	// If not public: reply that we don't do DMs
	if !toPublicOrFollowers {
		ib.logger.Info("Message we got is a DM")
		msg := ib.txt.WithVals("reply_no_dm.html", map[string]string{
			"moniker": moniker,
			"userUrl": senderInfo.Id,
		})
		go ib.messenger.SendMessageSync(ib.cfg.Birb.User, senderInfo.Inbox, msg,
			[]*MsgMention{{moniker, act.Actor}},
			[]string{act.Actor}, []string{},
			act.Object.Id)
		return
	}

	// Look for exactly 1 valid URL in message
	blogUrl := ib.getUrl(act.Object.Content)
	if blogUrl == "" {
		ib.logger.Info("No single URL found in message")
		msg := ib.txt.WithVals("reply_no_single_url.html", map[string]string{
			"moniker": moniker,
			"userUrl": senderInfo.Id,
		})
		go ib.messenger.SendMessageSync(ib.cfg.Birb.User, senderInfo.Inbox, msg,
			[]*MsgMention{{moniker, act.Actor}},
			[]string{shared.ActivityPublic}, []string{act.Actor, senderInfo.Followers},
			act.Object.Id)
		return
	}

	go ib.handleSiteRequest(senderInfo, act, moniker, blogUrl)

	return
}

func (ib *inbox) handleSiteRequest(senderInfo *dto.UserInfo, act dto.ActivityIn[dto.Note], moniker, blogUrl string) {

	acct, _, err := ib.fdfol.GetAccountForFeed(blogUrl)

	if acct == nil {
		ib.logger.Infof("Could not create/retrieve account for site: %s: %v", blogUrl, err)
		msg := ib.txt.WithVals("reply_site_not_found.html", map[string]string{
			"moniker": moniker,
			"userUrl": senderInfo.Id,
		})
		go ib.messenger.SendMessageSync(ib.cfg.Birb.User, senderInfo.Inbox, msg,
			[]*MsgMention{{moniker, act.Actor}},
			[]string{shared.ActivityPublic}, []string{act.Actor, senderInfo.Followers},
			act.Object.Id)
		return
	}

	ib.logger.Infof("Account for site created/retrieved: %s -> %s", blogUrl, acct.Handle)
	accountMoniker := shared.MakeFullMoniker(ib.cfg.Host, acct.Handle)
	accountUrl := ib.idb.UserUrl(acct.Handle)
	msg := ib.txt.WithVals("reply_got_feed.html", map[string]string{
		"userHandle":     senderInfo.PreferredUserName,
		"userUrl":        senderInfo.Id,
		"accountName":    acct.Name,
		"accountMoniker": "@" + acct.Handle,
		"accountUrl":     accountUrl,
	})
	go ib.messenger.SendMessageSync(ib.cfg.Birb.User, senderInfo.Inbox, msg,
		[]*MsgMention{{moniker, act.Actor}, {accountMoniker, accountUrl}},
		[]string{shared.ActivityPublic}, []string{act.Actor, senderInfo.Followers},
		act.Object.Id)

	// @birb@rss-parrot.zydeo.net https://r4inee.wordpress.com
	// @birb@rss-parrot.zydeo.net https://soatok.blog/b/
	// @birb@rss-parrot.zydeo.net https://magazine.sebastianraschka.com/
	// @birb@rss-parrot.zydeo.net https://mastodon.social/@zydeobor
}

func (ib *inbox) getUrl(content string) string {

	pol := bluemonday.StrictPolicy()
	plain := pol.Sanitize(content)
	matches := ib.reHttps.FindAllString(plain, -1)
	// Looking for exactly one valid Url
	res := ""
	for _, str := range matches {
		_, err := url.Parse(str)
		if err != nil {
			continue
		}
		if res != "" {
			return ""
		}
		res = str
	}
	return res
}
