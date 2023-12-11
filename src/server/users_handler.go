package server

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"rss_parrot/logic"
)

type UsersHandler struct {
	ud logic.IUserDirectory
}

func NewUsersHandler(ud logic.IUserDirectory) *UsersHandler {
	return &UsersHandler{ud}
}

func (*UsersHandler) Def() (string, string) {
	return "GET", "/users/{user}"
}

func (h *UsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Printf("Users: Request received")
	userName := mux.Vars(r)["user"]

	userInfo := h.ud.GetUserInfo(userName)

	if userInfo == nil {
		log.Printf("Users: No such user: '%s'", userName)
		http.Error(w, notFoundStr, http.StatusNotFound)
		return
	}

	writeResponse(w, userInfo)
}
