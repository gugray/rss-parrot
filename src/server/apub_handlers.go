package server

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"regexp"
	"rss_parrot/logic"
	"strconv"
)

// Groups together the handlers needed to implement an ActivityPub server.
type apubHandlerGroup struct {
	sender     logic.IActivitySender
	wfing      logic.IWebfinger
	udir       logic.IUserDirectory
	obox       logic.IOutbox
	reResource *regexp.Regexp
}

func NewApubHandlerGroup(
	sender logic.IActivitySender,
	wfing logic.IWebfinger,
	udir logic.IUserDirectory,
	obox logic.IOutbox,
) IHandlerGroup {
	res := apubHandlerGroup{
		sender: sender,
		wfing:  wfing,
		udir:   udir,
		obox:   obox,
	}
	res.reResource = regexp.MustCompile("^acct:([^@]+)@([^@]+)$")
	return &res
}

func (hg *apubHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"GET", "/.well-known/webfinger", func(w http.ResponseWriter, r *http.Request) { hg.getWebfinger(w, r) }},
		{"GET", "/users/{user}", func(w http.ResponseWriter, r *http.Request) { hg.getUsers(w, r) }},
		{"GET", "/users/{user}/outbox", func(w http.ResponseWriter, r *http.Request) { hg.getOutbox(w, r) }},
	}
}

func (hg *apubHandlerGroup) getOutbox(w http.ResponseWriter, r *http.Request) {

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

	summary := hg.obox.GetOutboxSummary(userName)

	writeResponse(w, summary)
}

func (hg *apubHandlerGroup) getUsers(w http.ResponseWriter, r *http.Request) {

	log.Printf("Users: Request received")
	userName := mux.Vars(r)["user"]

	userInfo := hg.udir.GetUserInfo(userName)

	if userInfo == nil {
		log.Printf("Users: No such user: '%s'", userName)
		http.Error(w, notFoundStr, http.StatusNotFound)
		return
	}

	writeResponse(w, userInfo)
}

func (hg *apubHandlerGroup) getWebfinger(w http.ResponseWriter, r *http.Request) {

	log.Printf("Webfinger: Request received")

	resourceParam := r.URL.Query().Get("resource")
	groups := hg.reResource.FindStringSubmatch(resourceParam)
	if groups == nil {
		log.Printf("Webfinger: Invalid request; 'resource' param is '%s'", resourceParam)
		http.Error(w, invalidRequestStr, http.StatusBadRequest)
		return
	}
	user, instance := groups[1], groups[2]

	resp := hg.wfing.MakeResponse(user, instance)

	if resp == nil {
		log.Printf("Webfinger: No such resource; 'resource' param is '%s'", resourceParam)
		http.Error(w, notFoundStr, http.StatusNotFound)
		return
	}

	writeResponse(w, resp)
}
