package test

import (
	"go.uber.org/mock/gomock"
	"rss_parrot/test/mocks"
)

func stubLogger(mockLogger *mocks.MockILogger) {
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

func stubTexts(mockTexts *mocks.MockITexts) {
	mockTexts.EXPECT().WithVals(gomock.Any(), gomock.Any()).
		DoAndReturn(func(id string, vals map[string]string) string {
			return dummyTextWithVals(id, vals)
		}).AnyTimes()
}

func dummyTextWithVals(id string, vals map[string]string) string {
	res := id
	for k, v := range vals {
		res += "\n" + k + "\t" + v
	}
	return res
}

func stubMetrics(mockMetrics *mocks.MockIMetrics) {
	mockMetrics.EXPECT().TotalFollowers(gomock.Any()).AnyTimes()
}
