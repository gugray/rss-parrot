package server

import (
	"log"
	"net/http"
	"regexp"
	"rss_parrot/logic"
)

type WebfingerHandler struct {
	wf         *logic.Webfinger
	reResource *regexp.Regexp
}

func NewWebfingerHandler(wf *logic.Webfinger) *WebfingerHandler {
	reResource := regexp.MustCompile("^acct:([^@]+)@([^@]+)$")
	return &WebfingerHandler{
		wf,
		reResource,
	}
}

func (*WebfingerHandler) Def() (string, string) {
	return "GET", "/.well-known/webfinger"
}

func (h *WebfingerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Printf("Webfinger: Request received")

	resourceParam := r.URL.Query().Get("resource")
	groups := h.reResource.FindStringSubmatch(resourceParam)
	if groups == nil {
		log.Printf("Webfinger: Invalid request; 'resource' param is '%s'", resourceParam)
		http.Error(w, invalidRequestStr, http.StatusBadRequest)
		return
	}
	user, instance := groups[1], groups[2]

	resp := h.wf.MakeResponse(user, instance)

	if resp == nil {
		log.Printf("Webfinger: No such resource; 'resource' param is '%s'", resourceParam)
		http.Error(w, notFoundStr, http.StatusNotFound)
		return
	}

	writeResponse(w, resp)
}
