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
}

func makeInboxHarness(ctrl *gomock.Controller) *inboxHarness {
	res := inboxHarness{
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
	}
	stubLogger(res.mockLogger)
	stubMetrics(res.mockMetrics)
	res.mockRepo.EXPECT().GetFeedFollowerCount().Return(0, nil).AnyTimes()
	return &res
}

const contentBirbNoUrl = `<p><span class=\"h-card\" translate=\"no\"><a href=\"https://rss-parrot.zydeo.net/u/birb\" class=\"u-url mention\">@<span>birb</span></a></span> Henlo</p>`

func Test_BirbMentioned_NoUrl(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := makeInboxHarness(ctrl)
	inbox := logic.NewInbox(h.cfg, h.mockLogger, h.mockRepo, h.mockTexts, h.mockMetrics, h.mockUDir,
		h.mockKeyStore, h.mockSender, h.mockMessenger, h.mockFF)

	sender := makeCallerUserInfo(callerHost, callerName, callerPubKey1)
	birbUrl := fmt.Sprintf("https://%s/u/%s", h.cfg.Host, h.cfg.Birb.User)
	birbMoniker := fmt.Sprintf("@%s@%s", h.cfg.Birb.User, h.cfg.Host)
	tags := `[{"type":"Mention","href": "` + birbUrl + `","name": "` + birbMoniker + `"}]`
	bodyBytes := makeCreateNote(callerHost, callerName, contentBirbNoUrl,
		[]string{birbUrl},
		[]string{},
		tags,
	)
	var act dto.ActivityInBase
	if err := json.Unmarshal(bodyBytes, &act); err != nil {
		panic(err)
	}

	h.mockRepo.EXPECT().MarkActivityHandled(gomock.Eq(act.Id), gomock.Any()).Return(false, nil)

	h.mockTexts.EXPECT().WithVals("reply_no_single_url.html", gomock.Any()).
		DoAndReturn(func(id string, vals map[string]string) string {
			return dummyTextWithVals(id, vals)
		})

	h.mockMessenger.EXPECT().SendMessageAsync(
		gomock.Eq(birbName),
		gomock.Eq(sender.Inbox),
		gomock.Any(), // Don't verfy message content
		gomock.Any(), // TODO: Check there is one mention, for sender
		gomock.Cond(strSliceMatch([]string{act.Actor})), // to: sender
		gomock.Cond(strSliceMatch([]string{})),          // cc: none
		gomock.Eq(act.Id))

	inbox.HandleCreateNote(act, sender, bodyBytes)
}
