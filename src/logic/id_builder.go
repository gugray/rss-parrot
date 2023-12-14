package logic

import (
	"fmt"
	"strconv"
)

type idBuilder struct {
	host string
}

func (idb *idBuilder) SharedInbox() string {
	return fmt.Sprintf("https://%s/inbox", idb.host)
}

func (idb *idBuilder) UserProfile(user string) string {
	return fmt.Sprintf("https://%s/@%s", idb.host, user)
}

func (idb *idBuilder) UserUrl(user string) string {
	return fmt.Sprintf("https://%s/users/%s", idb.host, user)
}

func (idb *idBuilder) UserKeyId(user string) string {
	return fmt.Sprintf("https://%s/users/%s#main-key", idb.host, user)
}

func (idb *idBuilder) UserInbox(user string) string {
	return fmt.Sprintf("https://%s/users/%s/inbox", idb.host, user)
}

func (idb *idBuilder) UserOutbox(user string) string {
	return fmt.Sprintf("https://%s/users/%s/outbox", idb.host, user)
}

func (idb *idBuilder) UserFollowing(user string) string {
	return fmt.Sprintf("https://%s/users/%s/following", idb.host, user)
}

func (idb *idBuilder) UserFollowers(user string) string {
	return fmt.Sprintf("https://%s/users/%s/followers", idb.host, user)
}

func (idb *idBuilder) UserStatus(user string, id uint64) string {
	idStr := strconv.FormatUint(id, 10)
	return fmt.Sprintf("https://%s/users/%s/status/%s", idb.host, user, idStr)
}

func (idb *idBuilder) UserStatusActivity(user string, id uint64) string {
	idStr := strconv.FormatUint(id, 10)
	return fmt.Sprintf("https://%s/users/%s/status/%s/activity", idb.host, user, idStr)
}
