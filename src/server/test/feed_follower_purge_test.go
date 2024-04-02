package test

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"rss_parrot/dal"
	"rss_parrot/logic"
	"rss_parrot/shared"
	"rss_parrot/test/mocks"
	"testing"
	"time"
)

type tootExtract struct {
	postTime     time.Time
	postGuidHash int64
}

type feedFollowerHarness struct {
	cfg              *shared.Config
	mockLogger       *mocks.MockILogger
	mockUserAgent    *mocks.MockIUserAgent
	mockRepo         *mocks.MockIRepo
	mockBlockedFeeds *mocks.MockIBlockedFeeds
	mockMessenger    *mocks.MockIMessenger
	mockTexts        *mocks.MockITexts
	mockKeyStore     *mocks.MockIKeyStore
	mockMetrics      *mocks.MockIMetrics
}

func setupFeedFollowerTest(t *testing.T) (*gomock.Controller, *feedFollowerHarness, logic.IFeedFollower) {

	ctrl := gomock.NewController(t)

	h := &feedFollowerHarness{
		cfg:              &shared.Config{},
		mockLogger:       mocks.NewMockILogger(ctrl),
		mockUserAgent:    mocks.NewMockIUserAgent(ctrl),
		mockRepo:         mocks.NewMockIRepo(ctrl),
		mockBlockedFeeds: mocks.NewMockIBlockedFeeds(ctrl),
		mockMessenger:    mocks.NewMockIMessenger(ctrl),
		mockTexts:        mocks.NewMockITexts(ctrl),
		mockKeyStore:     mocks.NewMockIKeyStore(ctrl),
		mockMetrics:      mocks.NewMockIMetrics(ctrl),
	}
	setupDummyLogger(h.mockLogger)
	setupDummyMetrics(h.mockMetrics)

	h.mockRepo.EXPECT().GetTotalPostCount().Return(uint(0), nil).AnyTimes()

	ff := logic.NewFeedFollower(h.cfg, h.mockLogger, h.mockUserAgent, h.mockRepo,
		h.mockBlockedFeeds, h.mockMessenger, h.mockTexts, h.mockKeyStore, h.mockMetrics)

	return ctrl, h, ff
}

func extractsToToots(postExtracts []tootExtract) []*dal.Toot {
	var res []*dal.Toot
	for _, e := range postExtracts {
		post := dal.Toot{
			PostGuidHash: e.postGuidHash,
			TootedAt:     e.postTime,
		}
		res = append(res, &post)
	}
	return res
}

func test_Feed_Follower_Purge_Old_Posts(t *testing.T,
	postExtracts []tootExtract, fromBefore *time.Time, minCount int) {

	// Set up inbox, harness, shared dummies
	ctrl, h, ff := setupFeedFollowerTest(t)
	defer ctrl.Finish()

	// No accounts to check: this will keep feed follower's update check loop quiet
	h.mockRepo.EXPECT().GetAccountToCheck(gomock.Any()).Return(nil, 0, nil).AnyTimes()

	acct := dal.Account{
		Id:     17,
		Handle: "some.site.com.feed",
	}

	h.mockRepo.EXPECT().GetTootExtracts(gomock.Eq(acct.Id)).Return(extractsToToots(postExtracts), nil).Times(1)
	if fromBefore != nil {
		h.mockRepo.EXPECT().
			PurgePostsAndToots(gomock.Eq(acct.Id), gomock.Eq(*fromBefore)).
			Return(nil).Times(1)
	}

	// Purge items beyond minCount that are older than 2 days
	err := ff.PurgeOldPosts(&acct, minCount, 2)
	assert.Nil(t, err)
}

func Test_Feed_Follower_Purge_Old_Posts_Scenarios(t *testing.T) {
	now := time.Now().UTC()
	tootExtracts := []tootExtract{
		{now.Add(-3 * time.Hour), int64(getNextId())},
		{now.Add(-52 * time.Hour), int64(getNextId())},
		{now.Add(-1 * time.Hour), int64(getNextId())},
		{now.Add(-49 * time.Hour), int64(getNextId())},
		{now.Add(-2 * time.Hour), int64(getNextId())},
	}
	test_Feed_Follower_Purge_Old_Posts(t, tootExtracts, nil, 5)
	test_Feed_Follower_Purge_Old_Posts(t, tootExtracts, &tootExtracts[1].postTime, 4)
	test_Feed_Follower_Purge_Old_Posts(t, tootExtracts, &tootExtracts[3].postTime, 3)
	test_Feed_Follower_Purge_Old_Posts(t, tootExtracts, &tootExtracts[3].postTime, 2)
}
