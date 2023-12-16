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
	cfg        *shared.Config
	logger     shared.ILogger
	repo       dal.IRepo
	keyHandler IKeyHandler
	sender     IActivitySender
	idb        shared.IdBuilder
}

func NewBroadcaster(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	keyHandler IKeyHandler,
	sender IActivitySender,
) IBroadcaster {
	return &broadcaster{
		cfg:        cfg,
		logger:     logger,
		repo:       repo,
		keyHandler: keyHandler,
		sender:     sender,
		idb:        shared.IdBuilder{cfg.Host},
	}
}

func (b *broadcaster) Broadcast(user, published, message string) error {

	followers, err := b.repo.GetFollowers(user)
	if err != nil {
		return err
	}

	// Collect distinct shared inboxes
	inboxes := make(map[string]struct{})
	for _, f := range followers {
		if _, exists := inboxes[f.SharedInbox]; !exists {
			inboxes[f.SharedInbox] = struct{}{}
		}
	}

	for addr := range inboxes {
		err = b.sendToInbox(addr, user, published, message)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *broadcaster) sendToInbox(addr, user, published, message string) error {

	b.logger.Infof("Sending to inbox: %s", addr)

	privKey, err := b.keyHandler.GetPrivKey(user)
	if err != nil {
		return err
	}

	id := b.repo.GetNextId()
	note := &dto.Note{
		Id:           b.idb.UserStatus(user, id),
		Type:         "Note",
		Published:    published,
		Summary:      nil,
		AttributedTo: b.idb.UserUrl(user),
		InReplyTo:    nil,
		Content:      message,
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:           []string{b.idb.UserFollowers(user)},
	}
	act := &dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      b.idb.UserStatusActivity(user, id),
		Type:    "Create",
		Actor:   b.idb.UserUrl(user),
		Object:  note,
	}

	b.sender.Send(privKey, user, addr, act)

	return nil
}
