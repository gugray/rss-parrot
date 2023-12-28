package logic

import (
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"time"
)

type IMessenger interface {
	SendMessageSync(byUser string, toInbox, msg string, mentions []*MsgMention, to, cc []string, inReplyTo string)
	EnqueueBroadcast(user string, statusId string, tootedAt time.Time, msg string) error
}

type MsgMention struct {
	Moniker string
	UserUrl string
}

const maxParallelSends = 5
const tootLoopIdleWakeSec = 5

type messenger struct {
	cfg             *shared.Config
	logger          shared.ILogger
	repo            dal.IRepo
	keyStore        IKeyStore
	sender          IActivitySender
	idb             shared.IdBuilder
	newTootsInQueue chan struct{}
	tqProgress      map[int]interface{}
}

func NewMessenger(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	keyStore IKeyStore,
	sender IActivitySender,
) IMessenger {
	m := messenger{
		cfg:      cfg,
		logger:   logger,
		repo:     repo,
		keyStore: keyStore,
		sender:   sender,
		idb:      shared.IdBuilder{cfg.Host},
	}
	m.newTootsInQueue = make(chan struct{})
	m.tqProgress = make(map[int]interface{})
	go m.tootQueueLoop()
	return &m
}

func (m *messenger) SendMessageSync(byUser string, toInbox, msg string,
	mentions []*MsgMention, to, cc []string, inReplyTo string,
) {
	published := time.Now().UTC().Format(time.RFC3339)
	var tags []dto.Tag
	for _, mention := range mentions {
		tags = append(tags, dto.Tag{Type: "Mention", Href: mention.UserUrl, Name: mention.Moniker})
	}
	var ptags *[]dto.Tag = nil
	if len(tags) != 0 {
		ptags = &tags
	}
	err := m.sendToInbox(byUser, to, cc, toInbox, &inReplyTo, published, msg, ptags)
	if err != nil {
		m.logger.Errorf("Failed to send message to inbox %s", toInbox)
	}
}

func (m *messenger) EnqueueBroadcast(user string, statusId string, tootedAt time.Time, msg string) error {

	followers, err := m.repo.GetFollowersByUser(user, true)
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

	// Create a queue item for each inbox
	for inboxUrl := range inboxes {
		err = m.repo.AddTootQueueItem(&dal.TootQueueItem{
			SendingUser: user,
			ToInbox:     inboxUrl,
			TootedAt:    tootedAt,
			StatusId:    statusId,
			Content:     msg,
		})
		if err != nil {
			return err
		}
	}

	go func() {
		m.newTootsInQueue <- struct{}{}
	}()

	return nil
}

func (m *messenger) tootQueueLoop() {

	tootSent := make(chan int)

	sendToots := func() {
		if len(m.tqProgress) >= maxParallelSends {
			return
		}
		maxId := -1
		for id := range m.tqProgress {
			maxId = max(maxId, id)
		}
		var err error
		var items []*dal.TootQueueItem
		items, err = m.repo.GetTootQueueItems(maxId, maxParallelSends-len(m.tqProgress))
		if err != nil {
			m.logger.Errorf("Failed to get toot queue items: %v", err)
			return
		}
		for _, item := range items {
			m.tqProgress[item.Id] = struct{}{}
			go m.sendQueuedToot(item, tootSent)
		}
	}

	removeSentToot := func(id int) {
		if err := m.repo.DeleteTootQueueItem(id); err != nil {
			m.logger.Errorf("Failed to remove sent toot from queue: %d: %v", id, err)
		}
		delete(m.tqProgress, id)
	}

	for {
		select {
		case <-m.newTootsInQueue:
			m.logger.Debug("New toots in queue")
			sendToots()
		case <-time.After(tootLoopIdleWakeSec * time.Second):
			m.logger.Debug("Toot queue idle loop")
			sendToots()
		case id := <-tootSent:
			m.logger.Debugf("Toot sent: %d", id)
			removeSentToot(id)
			sendToots()
		}
	}
}

func (m *messenger) sendQueuedToot(item *dal.TootQueueItem, tootSent chan int) {

	idb := shared.IdBuilder{m.cfg.Host}
	to := []string{shared.ActivityPublic}
	userFollowers := idb.UserFollowers(item.SendingUser)
	err := m.sendToInbox(
		item.SendingUser,
		to,
		[]string{userFollowers},
		item.ToInbox,
		nil,
		item.TootedAt.UTC().Format(time.RFC3339),
		item.Content,
		nil)
	if err != nil {
		m.logger.Errorf("Failed to send queued toot: %v", err)
	}

	tootSent <- item.Id
}

func (m *messenger) sendToInbox(byUser string, to, cc []string, toInbox string,
	inReplyTo *string, published, message string, tag *[]dto.Tag) error {

	m.logger.Infof("Sending to inbox: %s", toInbox)

	privKey, err := m.keyStore.GetPrivKey(byUser)
	if err != nil {
		return err
	}

	id := m.repo.GetNextId()
	note := &dto.Note{
		Id:           m.idb.UserStatus(byUser, id),
		Type:         "Note",
		Published:    published,
		Summary:      nil,
		AttributedTo: m.idb.UserUrl(byUser),
		InReplyTo:    inReplyTo,
		Content:      message,
		To:           to,
		Cc:           cc,
		Tag:          tag,
	}
	act := &dto.ActivityOut{
		Context: "https://www.w3.org/ns/activitystreams",
		Id:      m.idb.UserStatusActivity(byUser, id),
		Type:    "Create",
		Actor:   m.idb.UserUrl(byUser),
		To:      &to,
		Cc:      &cc,
		Object:  note,
	}

	m.sender.Send(privKey, byUser, toInbox, act)

	return nil
}
