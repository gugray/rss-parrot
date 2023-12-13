package logic

import (
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
)

type IBroadcaster interface {
	Broadcast(user string, published, message string) error
}

type broadcaster struct {
	cfg    *shared.Config
	logger shared.ILogger
	repo   dal.IRepo
	sender IActivitySender
}

func NewBroadcaster(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	sender IActivitySender,
) IBroadcaster {
	return &broadcaster{cfg, logger, repo, sender}
}

func (b *broadcaster) Broadcast(user, published, message string) error {

	// Collect distinct shared inboxes
	inboxes := make(map[string]struct{})
	followers := b.repo.GetFollowers()
	for _, f := range followers {
		if _, exists := inboxes[f.SharedInbox]; !exists {
			inboxes[f.SharedInbox] = struct{}{}
		}
	}

	for addr := range inboxes {
		b.sendToInbox(addr, user, published, message)
	}

	return nil
}

func (b *broadcaster) sendToInbox(addr, user, published, message string) {

	b.logger.Infof("Sending to inbox: %s", addr)

	cfgInstance := b.cfg.InstanceName
	id := b.repo.GetNextId()
	userUrl := fmtUserUrl(cfgInstance, user)
	note := &dto.Note{
		Id:           fmtUserStatus(cfgInstance, user, id),
		Type:         "Note",
		Published:    published,
		Summary:      nil,
		AttributedTo: userUrl,
		InReplyTo:    nil,
		Content:      message,
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:           []string{fmtUserFollowers(cfgInstance, user)},
	}
	act := &dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      fmtUserStatusActivity(cfgInstance, user, id),
		Type:    "Create",
		Actor:   userUrl,
		Object:  note,
	}

	b.sender.Send(addr, act)
}

//func() {
//	// Maps from shared inbox (one per instance) to list of users there
//	inboxToUsers := make(map[string][]string)
//
//	// Collect who we'll be addressing in each instance
//	followers := b.repo.GetFollowers()
//	for _, f := range followers {
//		var users []string
//		if val, exists := inboxToUsers[f.SharedInbox]; exists {
//			users = val
//		}
//		users = append(users, f.User)
//		inboxToUsers[f.SharedInbox] = users
//	}
//}
