package server

import (
	"context"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"net"
	"net/http"
	"rss_parrot/shared"
	"strconv"
)

func NewHTTPServer(cfg *shared.Config, logger shared.ILogger, lc fx.Lifecycle, router *mux.Router) *http.Server {
	addStr := ":" + strconv.FormatUint(uint64(cfg.ServicePort), 10)
	srv := &http.Server{Addr: addStr, Handler: router}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			logger.Printf("Starting HTTP server at %v", srv.Addr)
			go srv.Serve(listener)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Printf("Shutting down HTTP server")
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func NewMux(groups []IHandlerGroup, logger shared.ILogger) *mux.Router {
	router := mux.NewRouter()
	for _, group := range groups {
		subRouter := router.PathPrefix(group.Prefix()).Subrouter()
		authMW := group.AuthMW()
		subRouter.Use(authMW)
		for _, def := range group.GroupDefs() {
			subRouter.HandleFunc(def.pattern, def.handler).Methods(def.method)
		}
	}
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./www/")))
	//r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handleFallback(logger, w, r) })
	// TODO: Fix fallback so we get a log of missed requests
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handleFallback(logger, w, r) })
	return router
}

func handleFallback(logger shared.ILogger, w http.ResponseWriter, r *http.Request) {
	query := ""
	if r.URL.RawQuery != "" {
		query += "?" + r.URL.RawQuery
	}
	body := string(readBody(logger, w, r))
	logger.Infof("404 %s request: %s%s", r.Method, r.URL.Path, query)
	if body != "" {
		logger.Infof("BODY: %s", body)
	}
	http.Error(w, notFoundStr, http.StatusNotFound)
}
