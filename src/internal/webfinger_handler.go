package internal

import (
	"fmt"
	"log"
	"net/http"
	"rss_parrot/internal/writers"
)

type WebfingerHandlerConfig interface {
	GetBaseUrl() string
}

type WebfingerHandler struct {
	cfg WebfingerHandlerConfig
}

func NewWebfingerHandler(cfg WebfingerHandlerConfig) *WebfingerHandler {
	return &WebfingerHandler{cfg}
}

func (*WebfingerHandler) Pattern() string {
	return "/.well-known/webfinger"
}

func (*WebfingerHandler) Method() string {
	return "GET"
}

func (h *WebfingerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resourceParam := r.URL.Query().Get("resource")
	fmt.Println(resourceParam)
	json := writers.WriteWebfingerJson(h.cfg.GetBaseUrl(), "twilliability")
	if _, err := fmt.Fprintln(w, json); err != nil {
		log.Printf("Failed to handle request: %v\n", err)
	}

	//body, err := io.ReadAll(r.Body)
	//if err != nil {
	//	http.Error(w, "Internal server error", http.StatusInternalServerError)
	//	return
	//}
	//if _, err := fmt.Fprintf(w, "Hello, %s\n", body); err != nil {
	//	http.Error(w, "Internal server error", http.StatusInternalServerError)
	//	return
	//}
}
