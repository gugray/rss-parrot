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

type postExtract struct {
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

	ff := logic.NewFeedFollower(h.cfg, h.mockLogger, h.mockUserAgent, h.mockRepo,
		h.mockBlockedFeeds, h.mockMessenger, h.mockTexts, h.mockKeyStore, h.mockMetrics)

	return ctrl, h, ff
}

func extractsToPosts(postExtracts []postExtract) []*dal.FeedPost {
	var res []*dal.FeedPost
	for _, e := range postExtracts {
		post := dal.FeedPost{
			PostGuidHash: e.postGuidHash,
			PostTime:     e.postTime,
		}
		res = append(res, &post)
	}
	return res
}

func test_Feed_Follower_Purge_Old_Posts(t *testing.T,
	postExtracts []postExtract, hashesToDel []int64, minCount int) {

	// Set up inbox, harness, shared dummies
	ctrl, h, ff := setupFeedFollowerTest(t)
	defer ctrl.Finish()

	// No accounts to check: this will keep feed follower's update check loop quiet
	h.mockRepo.EXPECT().GetAccountToCheck(gomock.Any()).Return(nil, 0, nil).AnyTimes()

	acct := dal.Account{
		Id:     17,
		Handle: "some.site.com.feed",
	}

	h.mockRepo.EXPECT().GetPostsExtract(gomock.Eq(acct.Id)).Return(extractsToPosts(postExtracts), nil).Times(1)
	if len(hashesToDel) > 0 {
		h.mockRepo.EXPECT().
			PurgePostsAndToots(gomock.Eq(acct.Id), gomock.Cond(checkEqAsSet(hashesToDel))).
			Return(nil).Times(1)
	}

	// Purge items beyond minCount that are older than 2 days
	err := ff.PurgeOldPosts(&acct, minCount, 2)
	assert.Nil(t, err)
}

func Test_Feed_Follower_Purge_Old_Posts_Scenarios(t *testing.T) {
	now := time.Now().UTC()
	postExtracts := []postExtract{
		{now.Add(-3 * time.Hour), int64(getNextId())},
		{now.Add(-52 * time.Hour), int64(getNextId())},
		{now.Add(-1 * time.Hour), int64(getNextId())},
		{now.Add(-49 * time.Hour), int64(getNextId())},
		{now.Add(-2 * time.Hour), int64(getNextId())},
	}
	hashesToDel := []int64{}
	test_Feed_Follower_Purge_Old_Posts(t, postExtracts, hashesToDel, 5)
	hashesToDel = append(hashesToDel, postExtracts[1].postGuidHash)
	test_Feed_Follower_Purge_Old_Posts(t, postExtracts, hashesToDel, 4)
	hashesToDel = append(hashesToDel, postExtracts[3].postGuidHash)
	test_Feed_Follower_Purge_Old_Posts(t, postExtracts, hashesToDel, 3)
	test_Feed_Follower_Purge_Old_Posts(t, postExtracts, hashesToDel, 2)
}
