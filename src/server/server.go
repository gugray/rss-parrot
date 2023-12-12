package server

import (
	"context"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"log"
	"net"
	"net/http"
	"rss_parrot/config"
	"strconv"
)

func NewHTTPServer(cfg *config.Config, lc fx.Lifecycle, router *mux.Router) *http.Server {
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

func NewMux(groups []IHandlerGroup) *mux.Router {
	r := mux.NewRouter()
	for _, group := range groups {
		for _, def := range group.GroupDefs() {
			r.HandleFunc(def.pattern, def.handler).Methods(def.method)
		}
	}
	return r
}
