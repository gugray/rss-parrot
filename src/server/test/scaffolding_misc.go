package test

import (
	"go.uber.org/mock/gomock"
	"rss_parrot/test/mocks"
	"strings"
)

func setupDummyLogger(mockLogger *mocks.MockILogger) {
	mockLogger.EXPECT().Error(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Errorf(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warnf(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Infof(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Debugf(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Printf(gomock.Any()).AnyTimes()
}

func setupFakeTexts(mockTexts *mocks.MockITexts) {
	mockTexts.EXPECT().WithVals(gomock.Any(), gomock.Any()).
		DoAndReturn(func(id string, vals map[string]string) string {
			return fakeTextWithVals(id, vals)
		}).AnyTimes()
}

func fakeTextWithVals(id string, vals map[string]string) string {
	res := id
	for k, v := range vals {
		res += "\n" + k + "\t" + v
	}
	return res
}

func setupDummyMetrics(mockMetrics *mocks.MockIMetrics) {
	mockMetrics.EXPECT().TotalFollowers(gomock.Any()).AnyTimes()
}

func checkStrSlice(items []string) func(x any) bool {
	res := func(x any) bool {
		slice, ok := x.([]string)
		if !ok {
			return false
		}
		if len(slice) != len(items) {
			return false
		}
		for i := 0; i < len(slice); i++ {
			if slice[i] != items[i] {
				return false
			}
		}
		return true
	}
	return res
}

func checkStartsWith(prefix string) func(x any) bool {
	res := func(x any) bool {
		str, ok := x.(string)
		if !ok {
			return false
		}
		return strings.HasPrefix(str, prefix)
	}
	return res
}
