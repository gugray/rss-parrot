package test

import (
	"encoding/json"
	"fmt"
	"go.uber.org/mock/gomock"
	"rss_parrot/dto"
	"rss_parrot/logic"
	"rss_parrot/shared"
	"rss_parrot/test/mocks"
	"testing"
	"time"
)

const callerHost = "stardust.community"
const callerName = "pixie"
const birbHost = "test-parrot.net"
const birbName = "birb"
const publicStream = "https://www.w3.org/ns/activitystreams#Public"

type Visibility int

const (
	vizPublic Visibility = iota
	vizUnlisted
	vizFollowers
	vizDirect
)

const contentBirbNoUrl = `<p><span class=\"h-card\" translate=\"no\"><a href=\"https://rss-parrot.zydeo.net/u/birb\" class=\"u-url mention\">@<span>birb</span></a></span> Henlo</p>`

type inboxHarness struct {
	cfg           *shared.Config
	mockLogger    *mocks.MockILogger
	mockRepo      *mocks.MockIRepo
	mockTexts     *mocks.MockITexts
	mockMetrics   *mocks.MockIMetrics
	mockUDir      *mocks.MockIUserDirectory
	mockKeyStore  *mocks.MockIKeyStore
	mockSender    *mocks.MockIActivitySender
	mockMessenger *mocks.MockIMessenger
	mockFF        *mocks.MockIFeedFollower
	sender        *dto.UserInfo
	birbUrl       string
	birbMoniker   string
}

func setupInboxTest(t *testing.T) (*gomock.Controller, *inboxHarness, logic.IInbox) {

	ctrl := gomock.NewController(t)

	h := &inboxHarness{
		cfg: &shared.Config{
			Host: birbHost,
			Birb: &shared.UserInfo{
				User:      birbName,
				Published: time.Now().UTC(),
				PubKey:    birbPubKey,
				PrivKey:   birbPrivKey,
			},
		},
		mockLogger:    mocks.NewMockILogger(ctrl),
		mockRepo:      mocks.NewMockIRepo(ctrl),
		mockTexts:     mocks.NewMockITexts(ctrl),
		mockMetrics:   mocks.NewMockIMetrics(ctrl),
		mockUDir:      mocks.NewMockIUserDirectory(ctrl),
		mockKeyStore:  mocks.NewMockIKeyStore(ctrl),
		mockSender:    mocks.NewMockIActivitySender(ctrl),
		mockMessenger: mocks.NewMockIMessenger(ctrl),
		mockFF:        mocks.NewMockIFeedFollower(ctrl),
		sender:        makeCallerUserInfo(callerHost, callerName, callerPubKey1),
	}
	h.birbUrl = fmt.Sprintf("https://%s/u/%s", h.cfg.Host, h.cfg.Birb.User)
	h.birbMoniker = fmt.Sprintf("@%s@%s", h.cfg.Birb.User, h.cfg.Host)

	setupDummyLogger(h.mockLogger)
	setupDummyMetrics(h.mockMetrics)
	h.mockRepo.EXPECT().GetFeedFollowerCount().Return(0, nil).AnyTimes()

	inbox := logic.NewInbox(h.cfg, h.mockLogger, h.mockRepo, h.mockTexts, h.mockMetrics, h.mockUDir,
		h.mockKeyStore, h.mockSender, h.mockMessenger, h.mockFF)

	return ctrl, h, inbox
}

func checkSenderMention(sender *dto.UserInfo, callerHost string) func(x any) bool {
	res := func(x any) bool {
		val, ok := x.([]*logic.MsgMention)
		if !ok || len(val) != 1 {
			return false
		}
		if val[0].UserUrl != sender.Id {
			return false
		}
		senderMoniker := fmt.Sprintf("@%s@%s", sender.Name, callerHost)
		if val[0].Moniker != senderMoniker {
			return false
		}
		return true
	}
	return res
}

func testBirbMentioned_NoUrl(t *testing.T, viz Visibility) {

	// Set up inbox, harness, shared dummies
	ctrl, h, inbox := setupInboxTest(t)
	defer ctrl.Finish()

	// Set up "Create Note" activity
	tags := `[{"type":"Mention","href": "` + h.birbUrl + `","name": "` + h.birbMoniker + `"}]`
	var actTo []string
	var actCC []string
	if viz == vizDirect {
		actTo = []string{h.birbUrl}
	} else if viz == vizFollowers {
		actTo = []string{h.sender.Followers}
		actCC = []string{h.birbUrl}
	} else if viz == vizUnlisted {
		actTo = []string{h.sender.Followers}
		actCC = []string{h.birbUrl, publicStream}
	} else if viz == vizPublic {
		actTo = []string{publicStream}
		actCC = []string{h.birbUrl, h.sender.Followers}
	}
	bodyBytes := makeCreateNote(callerHost, callerName, contentBirbNoUrl, actTo, actCC, tags)
	var act dto.ActivityInBase
	if err := json.Unmarshal(bodyBytes, &act); err != nil {
		panic(err)
	}

	// Expect inbox to check if activity has been handled (no)
	h.mockRepo.EXPECT().MarkActivityHandled(gomock.Eq(act.Id), gomock.Any()).Return(false, nil)

	// Expected response content
	h.mockTexts.EXPECT().WithVals("reply_no_single_url.html", gomock.Any()).
		DoAndReturn(func(id string, vals map[string]string) string {
			return fakeTextWithVals(id, vals)
		})

	// Details of expected response message
	var respTo []string
	var respCC []string
	if viz == vizDirect {
		respTo = []string{h.sender.Id}
	} else {
		respTo = []string{publicStream}
		respCC = []string{h.sender.Id, h.sender.Followers}
	}
	h.mockMessenger.EXPECT().SendMessageAsync(
		gomock.Eq(birbName),
		gomock.Eq(h.sender.Inbox),
		gomock.Any(), // don't verify message content; we have expectation on mockTexts
		gomock.Cond(checkSenderMention(h.sender, callerHost)), // there is one mention, for sender
		gomock.Cond(checkStrSlice(respTo)),
		gomock.Cond(checkStrSlice(respCC)),
		gomock.Eq(act.Id))

	// Execute
	inbox.HandleCreateNote(act, h.sender, bodyBytes)
}

func Test_BirbMentioned_Direct_NoUrl(t *testing.T) {
	testBirbMentioned_NoUrl(t, vizDirect)
}

func Test_BirbMentioned_Followers_NoUrl(t *testing.T) {
	testBirbMentioned_NoUrl(t, vizFollowers)
}

func Test_BirbMentioned_Unlisted_NoUrl(t *testing.T) {
	testBirbMentioned_NoUrl(t, vizUnlisted)
}

func Test_BirbMentioned_Public_NoUrl(t *testing.T) {
	testBirbMentioned_NoUrl(t, vizPublic)
}
