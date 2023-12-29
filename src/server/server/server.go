package server

import (
	"context"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"net"
	"net/http"
	"rss_parrot/shared"
	"strconv"
	"strings"
)

const assetsDir = "/assets/"
const strCacheControlHdr = "Cache-Control"
const strCacheControlNoChacheVal = "max-age=31536000, immutable"

var staticFS = http.FileServer(http.Dir("./www/"))

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
		subRouter.Use(noCacheMW)
		subRouter.Use(authMW)
		for _, def := range group.GroupDefs() {
			subRouter.HandleFunc(def.pattern, def.handler).Methods("OPTIONS", def.method)
		}
	}
	// Static files with error logging
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleStatic(logger, w, r)
	})
	return router
}

func noCacheMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(strCacheControlHdr, strCacheControlNoChacheVal)
		next.ServeHTTP(w, r)
	})
}

func handleStatic(logger shared.ILogger, w http.ResponseWriter, r *http.Request) {

	logNonOK := func(code int) {
		query := ""
		if r.URL.RawQuery != "" {
			query += "?" + r.URL.RawQuery
		}
		logger.Infof("%s request had status %d: %s%s", r.Method, code, r.URL.Path, query)
	}

	isAsset := strings.HasPrefix(r.URL.Path, assetsDir)
	if r.Method == "GET" && r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
		logNonOK(403)
		http.Error(w, dirListNotAllowed, 403)
		return
	}

	cw := staticFileResponseWriter{w, http.StatusOK, isAsset}
	staticFS.ServeHTTP(&cw, r)
	if cw.statusCode >= 400 {
		logNonOK(cw.statusCode)
	}
}

type staticFileResponseWriter struct {
	http.ResponseWriter
	statusCode int
	cached     bool
}

func (lrw *staticFileResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	if lrw.cached {
		lrw.ResponseWriter.Header().Set(strCacheControlHdr, strCacheControlNoChacheVal)
	} else {
		lrw.ResponseWriter.Header().Set(strCacheControlHdr, "no-cache, no-store, must-revalidate")
		lrw.ResponseWriter.Header().Set("Pragma", "no-cache")
		lrw.ResponseWriter.Header().Set("Expires", "0")
	}
	lrw.ResponseWriter.WriteHeader(code)
}
