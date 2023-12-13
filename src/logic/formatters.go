package logic

import (
	"fmt"
	"strconv"
)

func fmtUserUrl(baseUrl, user string) string {
	return fmt.Sprintf("https://%s/users/%s", baseUrl, user)
}

func fmtUserStatus(baseUrl, user string, id uint64) string {
	idStr := strconv.FormatUint(id, 10)
	return fmt.Sprintf("https://%s/users/%s/status/%s", baseUrl, user, idStr)
}

func fmtUserStatusActivity(baseUrl, user string, id uint64) string {
	idStr := strconv.FormatUint(id, 10)
	return fmt.Sprintf("https://%s/users/%s/status/%s/activity", baseUrl, user, idStr)
}

func fmtUserFollowers(baseUrl, user string) string {
	return fmt.Sprintf("https://%s/users/%s/followers", baseUrl, user)
}
