package internal

import (
	"context"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"log"
	"net"
	"net/http"
	"strconv"
)

type ServerConfig interface {
	GetServicePort() uint
}

type Route interface {
	http.Handler
	Pattern() string
	Method() string
}

func NewHTTPServer(cfg ServerConfig, lc fx.Lifecycle, router *mux.Router) *http.Server {
	addStr := ":" + strconv.FormatUint(uint64(cfg.GetServicePort()), 10)
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
		r.Handle(route.Pattern(), route).Methods(route.Method())
	}
	return r
}
