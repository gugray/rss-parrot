package server

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"rss_parrot/logic"
	"strconv"
)

type OutboxHandler struct {
	ob logic.IOutbox
}

func NewOutboxHandler(ob logic.IOutbox) *OutboxHandler {
	return &OutboxHandler{ob}
}

func (*OutboxHandler) Def() (string, string) {
	return "GET", "/users/{user}/outbox"
}

func (h *OutboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var err error
	_ = err // But why?
	log.Printf("Outbox: Request received")
	userName := mux.Vars(r)["user"]
	pageParam := r.URL.Query().Get("page")
	minId := -1
	maxId := -1
	if r.URL.Query().Has("min_id") {
		minId, err = strconv.Atoi(r.URL.Query().Get("min_id"))
		// TODO: Handle error / bad request
	}
	if r.URL.Query().Has("max_id") {
		maxId, err = strconv.Atoi(r.URL.Query().Get("max_id"))
		// TODO: Handle error / bad request
	}
	_ = minId
	_ = maxId

	if pageParam == "true" { // TODO: page posts
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}

	summary := h.ob.GetOutboxSummary(userName)

	writeResponse(w, summary)
}
