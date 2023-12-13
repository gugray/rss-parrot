package logic

import (
	"fmt"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/shared"
	"strings"
)

type IOutbox interface {
	GetOutboxSummary(user string) *dto.OutboxSummary
}

type outbox struct {
	cfg  *shared.Config
	repo dal.IRepo
}

func NewOutbox(cfg *shared.Config, repo dal.IRepo) IOutbox {
	return &outbox{cfg, repo}
}

func (ob *outbox) GetOutboxSummary(user string) *dto.OutboxSummary {

	cfgInstance := ob.cfg.InstanceName
	cfgBirb := ob.cfg.BirbName
	if !strings.EqualFold(user, cfgBirb) {
		return nil
	}

	user = strings.ToLower(user)
	userId := fmt.Sprintf("https://%s/users/%s", cfgInstance, user)
	obUrl := fmt.Sprintf("%s/outbox", userId)

	resp := dto.OutboxSummary{
		Context:    "https://www.w3.org/ns/activitystreams",
		Id:         obUrl,
		Type:       "OrderedCollection",
		TotalItems: ob.repo.GetPostCount(),
		First:      fmt.Sprintf("%s?page=true", obUrl),
		Last:       fmt.Sprintf("%s?page=true&min_id=0", obUrl),
	}
	return &resp
}
