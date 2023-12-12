package server

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"regexp"
	"rss_parrot/dto"
	"rss_parrot/logic"
	"strconv"
)

// Groups together the handlers needed to implement an ActivityPub server.
type apubHandlerGroup struct {
	sender     logic.IActivitySender
	sigChecker logic.IHttpSigChecker
	wfing      logic.IWebfinger
	udir       logic.IUserDirectory
	obox       logic.IOutbox
	ibox       logic.IInbox
	reResource *regexp.Regexp
}

func NewApubHandlerGroup(
	sender logic.IActivitySender,
	sigChecker logic.IHttpSigChecker,
	wfing logic.IWebfinger,
	udir logic.IUserDirectory,
	obox logic.IOutbox,
	ibox logic.IInbox,
) IHandlerGroup {
	res := apubHandlerGroup{
		sender:     sender,
		sigChecker: sigChecker,
		wfing:      wfing,
		udir:       udir,
		obox:       obox,
		ibox:       ibox,
	}
	res.reResource = regexp.MustCompile("^acct:([^@]+)@([^@]+)$")
	return &res
}

func (hg *apubHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"GET", "/.well-known/webfinger", func(w http.ResponseWriter, r *http.Request) { hg.getWebfinger(w, r) }},
		{"GET", "/users/{user}", func(w http.ResponseWriter, r *http.Request) { hg.getUsers(w, r) }},
		{"GET", "/users/{user}/outbox", func(w http.ResponseWriter, r *http.Request) { hg.getOutbox(w, r) }},
		{"POST", "/users/{user}/inbox", func(w http.ResponseWriter, r *http.Request) { hg.postInbox(w, r) }},
		{"POST", "/inbox", func(w http.ResponseWriter, r *http.Request) { hg.postInbox(w, r) }},
	}
}

func (hg *apubHandlerGroup) postInbox(w http.ResponseWriter, r *http.Request) {

	var senderInfo *dto.UserInfo
	var sigOk bool
	if senderInfo, sigOk = hg.sigChecker.Check(w, r); !sigOk {
		return
	}

	var err error
	log.Printf("Inbox: POST request received: %s", r.URL.Path)
	userName := mux.Vars(r)["user"]
	bodyBytes := readBody(w, r)
	if bodyBytes == nil {
		return
	}

	// DBG
	log.Println(string(bodyBytes))

	// First, parse a rudimentary version of the activity to find out what it is
	var act dto.ActivityInBase
	if err = json.Unmarshal(bodyBytes, &act); err != nil {
		log.Printf("Invalid JSON in request body")
		http.Error(w, badRequestStr, http.StatusBadRequest)
		return
	}

	// Does signer match actor?
	if senderInfo.Id != act.Actor {
		log.Printf("Activity signed by %s, but actor is %s", senderInfo.Id, act.Actor)
		http.Error(w, unauthorizedStr, http.StatusUnauthorized)
	}

	// Handle different activities
	var badReq error
	if act.Type == "Follow" {
		badReq, err = hg.ibox.HandleFollow(userName, senderInfo, bodyBytes)
	}
	if badReq != nil {
		log.Printf("Invalid request: %v", badReq)
		http.Error(w, badRequestStr, http.StatusBadRequest) // TODO: return message in JSON
		return
	}

	writeResponse(w, "OK")
}

func (hg *apubHandlerGroup) getOutbox(w http.ResponseWriter, r *http.Request) {

	var err error
	_ = err // But why?
	log.Printf("Outbox: GET request received: %s", r.URL.Path)
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
		log.Printf("Received ?page=true; not handled yet")
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}

	summary := hg.obox.GetOutboxSummary(userName)

	writeResponse(w, summary)
}

func (hg *apubHandlerGroup) getUsers(w http.ResponseWriter, r *http.Request) {

	log.Printf("Users: GET request received: %s", r.URL.Path)
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

	log.Printf("Webfinger: GET request received")

	resourceParam := r.URL.Query().Get("resource")
	groups := hg.reResource.FindStringSubmatch(resourceParam)
	if groups == nil {
		log.Printf("Webfinger: Invalid request; 'resource' param is '%s'", resourceParam)
		http.Error(w, badRequestStr, http.StatusBadRequest)
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
