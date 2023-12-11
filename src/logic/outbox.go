package logic

import (
	"fmt"
	"rss_parrot/config"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"strings"
)

type IOutbox interface {
	GetOutboxSummary(user string) *dto.OutboxSummary
}

type Outbox struct {
	cfg  *config.Config
	repo dal.IRepo
}

func NewOutbox(cfg *config.Config, repo dal.IRepo) IOutbox {
	return &Outbox{cfg, repo}
}

func (ob *Outbox) GetOutboxSummary(user string) *dto.OutboxSummary {

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
		TotalItems: ob.repo.GetPostCount(),
		First:      fmt.Sprintf("%s?page=true", obUrl),
		Last:       fmt.Sprintf("%s?page=true&min_id=0", obUrl),
	}
	return &resp
}
