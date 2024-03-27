package test

import (
	"encoding/json"
	"fmt"
	"go.uber.org/mock/gomock"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/logic"
	"rss_parrot/shared"
	"rss_parrot/test/mocks"
	"strings"
	"sync"
	"testing"
	"time"
)

const callerHost = "stardust.community"
const callerName = "pixie"
const callerNameExtra = "ziggy"
const birbHost = "test-parrot.net"
const birbName = "birb"
const publicStream = "https://www.w3.org/ns/activitystreams#Public"
const requestedHost = "cute-animals.xyz"
const requestedPath = "blog"

type Visibility int

const (
	vizPublic Visibility = iota
	vizUnlisted
	vizFollowers
	vizDirect
)

type AtBirbMessageKind int

const (
	abmkNoUrl = iota
	abmkOneUrlGotFeed
	abmkOneUrlFeedError
	abmkOneUrlMastodonFeed
	abmkOneUrlBlockedFeed
	abmkOneUrlFeedOptedOut
)

type ReplyKind int

const (
	rkNotAReply = iota
	rkInReplyTo
)

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
	h.mockRepo.EXPECT().DeleteHandledActivities(gomock.Any()).AnyTimes()

	inbox := logic.NewInbox(h.cfg, h.mockLogger, h.mockRepo, h.mockTexts, h.mockMetrics, h.mockUDir,
		h.mockKeyStore, h.mockSender, h.mockMessenger, h.mockFF)

	return ctrl, h, inbox
}

func checkSenderMention(sender *dto.UserInfo, callerHost string, withParrot bool) func(x any) bool {
	res := func(x any) bool {
		val, ok := x.([]*logic.MsgMention)
		expectedLen := 1
		if withParrot {
			expectedLen = 2
		}
		if !ok || len(val) != expectedLen {
			return false
		}
		if val[0].UserUrl != sender.Id {
			return false
		}
		senderMoniker := fmt.Sprintf("@%s@%s", sender.Name, callerHost)
		if val[0].Moniker != senderMoniker {
			return false
		}
		if withParrot {
			acct := makeRequestedAccount()
			if val[1].UserUrl != acct.UserUrl {
				return false
			}
			accountMoniker := fmt.Sprintf("@%s@%s", acct.Handle, birbHost)
			if val[1].Moniker != accountMoniker {
				return false
			}
		}
		return true
	}
	return res
}

func makeRequestedAccount() *dal.Account {
	handle := requestedHost + "." + requestedPath
	handle = strings.ReplaceAll(handle, "/", ".")
	handle = strings.ReplaceAll(handle, "-", ".")
	return &dal.Account{
		Id:              1,
		CreatedAt:       time.Now().UTC(),
		UserUrl:         fmt.Sprintf("https://%s/u/%s", birbHost, handle),
		Handle:          handle,
		FeedName:        "Test feed",
		FeedSummary:     "Description of test feed",
		SiteUrl:         fmt.Sprintf("https://%s/%s", requestedHost, requestedPath),
		FeedUrl:         fmt.Sprintf("https://%s/%s/feed", requestedHost, requestedPath),
		FeedLastUpdated: time.Now(),
		NextCheckDue:    time.Now().Add(time.Hour * 6),
		PubKey:          accountPubKey1,
		ProfileImageUrl: "",
		HeaderImageUrl:  "",
	}
}

func testInbox_CreateNoteActivity(t *testing.T, viz Visibility, content string,
	repk ReplyKind, msgKind AtBirbMessageKind) {

	// Set up inbox, harness, shared dummies
	ctrl, h, inbox := setupInboxTest(t)
	defer ctrl.Finish()
	// Will need to wait on this due to expected calls in goroutines
	var wg sync.WaitGroup

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
	var inReplyTo *string = nil
	if repk == rkInReplyTo {
		statusId := fmt.Sprintf("https://%s/users/%s/status/170829", callerHost, callerNameExtra)
		inReplyTo = &statusId
	} else if repk != rkNotAReply {
		panic(fmt.Sprintf("Reply kind not impleneted: %v", repk))
	}
	bodyBytes := makeCreateNote(callerHost, callerName, content, actTo, actCC, inReplyTo, tags)
	var act dto.ActivityInBase
	if err := json.Unmarshal(bodyBytes, &act); err != nil {
		panic(err)
	}

	// Birb must do absolutely nothing if message is a reply.
	// Otherwise, this is what we expect to happen.
	if repk == rkNotAReply {
		// Expect inbox to check if activity has been handled (no)
		h.mockRepo.EXPECT().MarkActivityHandled(gomock.Eq(act.Id), gomock.Any()).Return(false, nil)

		// Expect inbox to get feed's account from FeedFollower
		if msgKind != abmkNoUrl {
			wg.Add(1)
			requestedUrl := fmt.Sprintf("https://%s/%s", requestedHost, requestedPath)
			h.mockFF.EXPECT().GetAccountForFeed(gomock.Eq(requestedUrl)).DoAndReturn(
				func(_ string) (*dal.Account, logic.FeedStatus, error) {
					defer wg.Done()
					if msgKind == abmkOneUrlGotFeed {
						return makeRequestedAccount(), logic.FsNew, nil
					} else if msgKind == abmkOneUrlFeedError {
						return nil, logic.FsError, fmt.Errorf("error getting feed")
					} else if msgKind == abmkOneUrlMastodonFeed {
						return nil, logic.FsMastodon, fmt.Errorf("feed is from mastodon")
					} else if msgKind == abmkOneUrlFeedOptedOut {
						return nil, logic.FsOptOut, fmt.Errorf("feed opted out")
					} else if msgKind == abmkOneUrlBlockedFeed {
						return nil, logic.FsBanned, fmt.Errorf("feed is blocked")
					} else {
						panic(fmt.Sprintf("Unhandled message kind: %v", msgKind))
					}
				},
			).Times(1)
		}

		// Expected response content
		var respTemplate string
		if msgKind == abmkNoUrl {
			respTemplate = "reply_no_single_url.html"
		} else if msgKind == abmkOneUrlGotFeed {
			respTemplate = "reply_got_feed.html"
		} else if msgKind == abmkOneUrlFeedError {
			respTemplate = "reply_site_not_found.html"
		} else if msgKind == abmkOneUrlMastodonFeed {
			respTemplate = "reply_feed_mastodon.html"
		} else if msgKind == abmkOneUrlFeedOptedOut {
			respTemplate = "reply_feed_optout.html"
		} else if msgKind == abmkOneUrlBlockedFeed {
			respTemplate = "reply_feed_banned.html"
		}
		h.mockTexts.EXPECT().WithVals(respTemplate, gomock.Any()).
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
		mentionWithParrot := msgKind == abmkOneUrlGotFeed
		wg.Add(1)
		h.mockMessenger.EXPECT().SendMessageAsync(
			gomock.Eq(birbName),
			gomock.Eq(h.sender.Inbox),
			gomock.Any(), // don't verify message content; we have expectation on mockTexts
			gomock.Cond(checkSenderMention(h.sender, callerHost, mentionWithParrot)),
			gomock.Cond(checkStrSlice(respTo)),
			gomock.Cond(checkStrSlice(respCC)),
			gomock.Eq(act.Id)).DoAndReturn(
			func(arg0, arg1, arg2, arg3, arg4, arg5, arg6 any) {
				wg.Done()
			}).Times(1)
	}

	// Execute
	inbox.HandleCreateNote(act, h.sender, bodyBytes)

	// Wait for async routines to finish
	//wg.Wait() // No timeout, needed when debugging test
	waitOnWG(t, &wg, time.Millisecond*500)
}

func waitOnWG(t *testing.T, wg *sync.WaitGroup, d time.Duration) {
	timer := time.After(d)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-timer:
		t.Errorf("Not all expected async methods have been called; gave up waiting.")
	}
}

const contentBirbNoUrl = `<p><span class=\"h-card\" translate=\"no\"><a href=\"https://rss-parrot.zydeo.net/u/birb\" class=\"u-url mention\">@<span>birb</span></a></span> Henlo</p>`
const contentBirbOneUrl = `<p><span class=\"h-card\" translate=\"no\"><a href=\"https://rss-parrot.zydeo.net/u/birb\" class=\"u-url mention\">@<span>birb</span></a></span> <a href=\"https://{{requested-url}}\" target=\"_blank\" rel=\"nofollow noopener noreferrer\" translate=\"no\"><span class=\"invisible\">https://</span><span class=\"\">{{requested-url}}</span><span class=\"invisible\"></span></a></p>`

// Message to birb with no URL
// -------------------------------------------
func TestInbox_BirbMentioned_NoUrl_Direct(t *testing.T) {
	testInbox_CreateNoteActivity(t, vizDirect, contentBirbNoUrl, rkNotAReply, abmkNoUrl)
}
func TestInbox_BirbMentioned_NoUrl_Followers(t *testing.T) {
	testInbox_CreateNoteActivity(t, vizFollowers, contentBirbNoUrl, rkNotAReply, abmkNoUrl)
}
func TestInbox_BirbMentioned_NoUrl_Unlisted(t *testing.T) {
	testInbox_CreateNoteActivity(t, vizUnlisted, contentBirbNoUrl, rkNotAReply, abmkNoUrl)
}
func TestInbox_BirbMentioned_NoUrl_Public(t *testing.T) {
	testInbox_CreateNoteActivity(t, vizPublic, contentBirbNoUrl, rkNotAReply, abmkNoUrl)
}

// Message to birb with no URL that's a reply
// -------------------------------------------
func TestInbox_BirbMentioned_Reply_NoUrl_Direct(t *testing.T) {
	testInbox_CreateNoteActivity(t, vizDirect, contentBirbNoUrl, rkInReplyTo, abmkNoUrl)
}
func TestInbox_BirbMentioned_Reply_NoUrl_Followers(t *testing.T) {
	testInbox_CreateNoteActivity(t, vizFollowers, contentBirbNoUrl, rkInReplyTo, abmkNoUrl)
}
func TestInbox_BirbMentioned_Reply_NoUrl_Unlisted(t *testing.T) {
	testInbox_CreateNoteActivity(t, vizUnlisted, contentBirbNoUrl, rkInReplyTo, abmkNoUrl)
}
func TestInbox_BirbMentioned_Reply_NoUrl_Public(t *testing.T) {
	testInbox_CreateNoteActivity(t, vizPublic, contentBirbNoUrl, rkInReplyTo, abmkNoUrl)
}

// Message to birb with one URL; feed found
// -------------------------------------------
func Test_BirbMentioned_OneUrl_GotFeed_Direct(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizDirect, content, rkNotAReply, abmkOneUrlGotFeed)
}
func Test_BirbMentioned_OneUrl_GotFeed_Followers(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizFollowers, content, rkNotAReply, abmkOneUrlGotFeed)
}
func Test_BirbMentioned_OneUrl_GotFeed_Unlisted(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizUnlisted, content, rkNotAReply, abmkOneUrlGotFeed)
}
func Test_BirbMentioned_OneUrl_GotFeed_Public(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizPublic, content, rkNotAReply, abmkOneUrlGotFeed)
}

// Message to birb with one URL; feed cannot be retrieved (error)
// -------------------------------------------
func Test_BirbMentioned_OneUrl_FeedNotFound_Direct(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizDirect, content, rkNotAReply, abmkOneUrlFeedError)
}
func Test_BirbMentioned_OneUrl_FeedNotFound_Followers(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizFollowers, content, rkNotAReply, abmkOneUrlFeedError)
}
func Test_BirbMentioned_OneUrl_FeedNotFound_Unlisted(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizUnlisted, content, rkNotAReply, abmkOneUrlFeedError)
}
func Test_BirbMentioned_OneUrl_FeedNotFound_Public(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizPublic, content, rkNotAReply, abmkOneUrlFeedError)
}

// Message to birb with one URL; feed is from Mastodon, not parroting
// -------------------------------------------
func Test_BirbMentioned_OneUrl_MastodonFeed_Direct(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizDirect, content, rkNotAReply, abmkOneUrlMastodonFeed)
}
func Test_BirbMentioned_OneUrl_MastodonFeed_Folowers(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizFollowers, content, rkNotAReply, abmkOneUrlMastodonFeed)
}
func Test_BirbMentioned_OneUrl_MastodonFeed_Unlisted(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizUnlisted, content, rkNotAReply, abmkOneUrlMastodonFeed)
}
func Test_BirbMentioned_OneUrl_MastodonFeed_Public(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizPublic, content, rkNotAReply, abmkOneUrlMastodonFeed)
}

// Message to birb with one URL; feed owner has opted out
// -------------------------------------------
func Test_BirbMentioned_OneUrl_FeedOptedOut_Direct(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizDirect, content, rkNotAReply, abmkOneUrlFeedOptedOut)
}
func Test_BirbMentioned_OneUrl_FeedOptedOut_Followers(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizFollowers, content, rkNotAReply, abmkOneUrlFeedOptedOut)
}
func Test_BirbMentioned_OneUrl_FeedOptedOut_Unlisted(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizUnlisted, content, rkNotAReply, abmkOneUrlFeedOptedOut)
}
func Test_BirbMentioned_OneUrl_FeedOptedOut_Public(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizPublic, content, rkNotAReply, abmkOneUrlFeedOptedOut)
}

// Message to birb with one URL; feed is blocked
// -------------------------------------------
func Test_BirbMentioned_OneUrl_BlockedFeed_Direct(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizDirect, content, rkNotAReply, abmkOneUrlBlockedFeed)
}
func Test_BirbMentioned_OneUrl_BlockedFeed_Followers(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizFollowers, content, rkNotAReply, abmkOneUrlBlockedFeed)
}
func Test_BirbMentioned_OneUrl_BlockedFeed_Unlisted(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizUnlisted, content, rkNotAReply, abmkOneUrlBlockedFeed)
}
func Test_BirbMentioned_OneUrl_BlockedFeed_Public(t *testing.T) {
	content := strings.ReplaceAll(contentBirbOneUrl, "{{requested-url}}", requestedHost+"/"+requestedPath)
	testInbox_CreateNoteActivity(t, vizPublic, content, rkNotAReply, abmkOneUrlBlockedFeed)
}
