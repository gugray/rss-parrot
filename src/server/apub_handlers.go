package server

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"regexp"
	"rss_parrot/dto"
	"rss_parrot/logic"
	"rss_parrot/shared"
	"strconv"
)

// Groups together the handlers needed to implement an ActivityPub server.
type apubHandlerGroup struct {
	logger     shared.ILogger
	sender     logic.IActivitySender
	sigChecker logic.IHttpSigChecker
	wfing      logic.IWebfinger
	udir       logic.IUserDirectory
	obox       logic.IOutbox
	ibox       logic.IInbox
	reResource *regexp.Regexp
}

func NewApubHandlerGroup(
	logger shared.ILogger,
	sender logic.IActivitySender,
	sigChecker logic.IHttpSigChecker,
	wfing logic.IWebfinger,
	udir logic.IUserDirectory,
	obox logic.IOutbox,
	ibox logic.IInbox,
) IHandlerGroup {
	res := apubHandlerGroup{
		logger:     logger,
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
	hg.logger.Info("Inbox: POST request received: %s", r.URL.Path)
	userName := mux.Vars(r)["user"]
	bodyBytes := readBody(hg.logger, w, r)
	if bodyBytes == nil {
		return
	}

	// DBG
	hg.logger.Debug(string(bodyBytes))

	// First, parse a rudimentary version of the activity to find out what it is
	var act dto.ActivityInBase
	if err = json.Unmarshal(bodyBytes, &act); err != nil {
		hg.logger.Info("Invalid JSON in request body")
		http.Error(w, badRequestStr, http.StatusBadRequest)
		return
	}

	// Does signer match actor?
	if senderInfo.Id != act.Actor {
		hg.logger.Warn("Activity signed by %s, but actor is %s", senderInfo.Id, act.Actor)
		http.Error(w, unauthorizedStr, http.StatusUnauthorized)
	}

	// Handle different activities
	var badReq error
	if act.Type == "Follow" {
		badReq, err = hg.ibox.HandleFollow(userName, senderInfo, bodyBytes)
	}
	if badReq != nil {
		hg.logger.Info("Invalid request: %v", badReq)
		http.Error(w, badRequestStr, http.StatusBadRequest) // TODO: return message in JSON
		return
	}

	writeResponse(hg.logger, w, "OK")
}

func (hg *apubHandlerGroup) getOutbox(w http.ResponseWriter, r *http.Request) {

	var err error
	_ = err // But why?
	hg.logger.Info("Outbox: GET request received: %s", r.URL.Path)
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
		hg.logger.Info("Received ?page=true; not handled yet")
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}

	summary := hg.obox.GetOutboxSummary(userName)

	writeResponse(hg.logger, w, summary)
}

func (hg *apubHandlerGroup) getUsers(w http.ResponseWriter, r *http.Request) {

	hg.logger.Info("Users: GET request received: %s", r.URL.Path)
	userName := mux.Vars(r)["user"]

	userInfo := hg.udir.GetUserInfo(userName)

	if userInfo == nil {
		hg.logger.Info("Users: No such user: '%s'", userName)
		http.Error(w, notFoundStr, http.StatusNotFound)
		return
	}

	writeResponse(hg.logger, w, userInfo)
}

func (hg *apubHandlerGroup) getWebfinger(w http.ResponseWriter, r *http.Request) {

	hg.logger.Info("Webfinger: GET request received")

	resourceParam := r.URL.Query().Get("resource")
	groups := hg.reResource.FindStringSubmatch(resourceParam)
	if groups == nil {
		hg.logger.Info("Webfinger: Invalid request; 'resource' param is '%s'", resourceParam)
		http.Error(w, badRequestStr, http.StatusBadRequest)
		return
	}
	user, instance := groups[1], groups[2]

	resp := hg.wfing.MakeResponse(user, instance)

	if resp == nil {
		hg.logger.Info("Webfinger: No such resource; 'resource' param is '%s'", resourceParam)
		http.Error(w, notFoundStr, http.StatusNotFound)
		return
	}

	writeResponse(hg.logger, w, resp)
}
