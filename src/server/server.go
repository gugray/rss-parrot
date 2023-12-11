package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"log"
	"net"
	"net/http"
	"rss_parrot/logic"
	"strconv"
)

const (
	internalErrorStr  = "Internal Server Error"
	invalidRequestStr = "Invalid Request"
	notFoundStr       = "Not Found"
)

type Route interface {
	http.Handler
	Def() (string, string)
}

func NewHTTPServer(cfg *logic.Config, lc fx.Lifecycle, router *mux.Router) *http.Server {
	addStr := ":" + strconv.FormatUint(uint64(cfg.ServicePort), 10)
	srv := &http.Server{Addr: addStr, Handler: router}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			log.Printf("Starting HTTP server at %v\n", srv.Addr)
			go srv.Serve(listener)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Shutting down HTTP server")
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func NewMux(routes []Route) *mux.Router {
	r := mux.NewRouter()
	for _, route := range routes {
		method, pattern := route.Def()
		r.Handle(pattern, route).Methods(method)
	}
	return r
}

func writeResponse(w http.ResponseWriter, resp interface{}) {
	var err error
	var respJson []byte
	if respJson, err = json.Marshal(resp); err != nil {
		log.Printf("Failed to serialize response: %v\n", err)
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}
	if _, err = fmt.Fprintln(w, string(respJson)); err != nil {
		log.Printf("Failed to write response: %v\n", err)
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}
}
