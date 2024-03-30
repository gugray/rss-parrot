package logic

import (
	"regexp"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"strconv"
	"time"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../test/mocks/mock_messenger.go -package mocks rss_parrot/logic IMessenger

type IMessenger interface {
	SendMessageAsync(byUser string, toInbox, msg string, mentions []*MsgMention, to, cc []string, inReplyTo string)
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
	metrics         IMetrics
	idb             shared.IdBuilder
	reStatusId      *regexp.Regexp
	newTootsInQueue chan struct{}
	tqProgress      map[int]interface{}
}

func NewMessenger(
	cfg *shared.Config,
	logger shared.ILogger,
	repo dal.IRepo,
	keyStore IKeyStore,
	sender IActivitySender,
	metrics IMetrics,
) IMessenger {

	m := messenger{
		cfg:      cfg,
		logger:   logger,
		repo:     repo,
		keyStore: keyStore,
		sender:   sender,
		metrics:  metrics,
		idb:      shared.IdBuilder{cfg.Host},
	}

	m.reStatusId = regexp.MustCompile("^https://[^/]+/u/[^/]+/status/([0-9]+)$")

	m.newTootsInQueue = make(chan struct{})
	m.tqProgress = make(map[int]interface{})
	go m.tootQueueLoop()

	return &m
}

func (m *messenger) SendMessageAsync(byUser string, toInbox, msg string,
	mentions []*MsgMention, to, cc []string, inReplyTo string,
) {
	go m.sendMessage(byUser, toInbox, msg, mentions, to, cc, inReplyTo)
}

func (m *messenger) sendMessage(byUser string, toInbox, msg string,
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
	id := m.repo.GetNextId()
	err := m.sendToInbox(byUser, id, to, cc, toInbox, &inReplyTo, published, msg, ptags)
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
		inboxName := f.SharedInbox
		if inboxName == "" {
			inboxName = f.UserInbox
		}
		if _, exists := inboxes[inboxName]; !exists {
			inboxes[inboxName] = struct{}{}
		}
	}

	if len(inboxes) == 0 {
		return nil
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
		var qlen int
		items, qlen, err = m.repo.GetTootQueueItems(maxId, maxParallelSends-len(m.tqProgress))
		if err != nil {
			m.logger.Errorf("Failed to get toot queue items: %v", err)
			return
		}
		m.metrics.TootQueueLength(qlen)
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

func (m *messenger) getIdVal(statusIdUrl string) uint64 {
	groups := m.reStatusId.FindStringSubmatch(statusIdUrl)
	if groups == nil {
		return m.repo.GetNextId()
	}
	idStr := groups[1]
	var idVal int64
	var err error
	if idVal, err = strconv.ParseInt(idStr, 10, 64); err != nil {
		return m.repo.GetNextId()
	}
	return uint64(idVal)
}

func (m *messenger) sendQueuedToot(item *dal.TootQueueItem, tootSent chan int) {

	var err error
	idb := shared.IdBuilder{m.cfg.Host}
	to := []string{shared.ActivityPublic}
	userFollowers := idb.UserFollowers(item.SendingUser)

	// This should never fail, but if it does, we just make up a new ID
	idVal := m.getIdVal(item.StatusId)

	err = m.sendToInbox(
		item.SendingUser,
		idVal,
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

	m.metrics.FeedTootSent()

	tootSent <- item.Id
}

func (m *messenger) sendToInbox(byUser string, idVal uint64, to, cc []string, toInbox string,
	inReplyTo *string, published, message string, tag *[]dto.Tag) error {

	m.logger.Infof("Sending to inbox: %s", toInbox)

	privKey, err := m.keyStore.GetPrivKey(byUser)
	if err != nil {
		return err
	}

	note := &dto.Note{
		Id:           m.idb.UserStatus(byUser, idVal),
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
		Id:      m.idb.UserStatusActivity(byUser, idVal),
		Type:    "Create",
		Actor:   m.idb.UserUrl(byUser),
		To:      &to,
		Cc:      &cc,
		Object:  note,
	}

	m.sender.Send(privKey, byUser, toInbox, act)

	return nil
}
