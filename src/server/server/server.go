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

const assetsDir = "/assets"
const strCacheControlHdr = "Cache-Control"

var staticFS = http.FileServer(http.Dir("./www/"))

func NewHTTPServer(cfg *shared.Config, logger shared.ILogger, lc fx.Lifecycle, router *mux.Router) *http.Server {
	addStr := ":" + strconv.FormatUint(uint64(cfg.ServicePort), 10)
	srv := &http.Server{Addr: addStr, Handler: trimSlashHandler(router)}
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

func trimSlashHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, assetsDir) {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		}
		next.ServeHTTP(w, r)
	})
}

func NewMux(groups []IHandlerGroup, logger shared.ILogger) *mux.Router {
	router := mux.NewRouter()
	for _, group := range groups {
		subRouter := router.PathPrefix(group.Prefix()).Subrouter()
		authMW := group.AuthMW()
		subRouter.Use(noCacheMW)
		subRouter.Use(authMW)
		for _, def := range group.GroupDefs() {
			if def.pattern == rootPlacholder {
				router.HandleFunc("/", def.handler).Methods("OPTIONS", def.method)
			} else {
				subRouter.HandleFunc(def.pattern, def.handler).Methods("OPTIONS", def.method)
			}
		}
	}
	// Static files with error logging
	router.PathPrefix(assetsDir).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleStatic(logger, w, r)
	})
	return router
}

func noCacheMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(strCacheControlHdr, "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

func handleStatic(logger shared.ILogger, w http.ResponseWriter, r *http.Request) {

	// TODO: pretty 404 and error

	logNonOK := func(code int) {
		query := ""
		if r.URL.RawQuery != "" {
			query += "?" + r.URL.RawQuery
		}
		logger.Infof("%s request had status %d: %s%s", r.Method, code, r.URL.Path, query)
	}

	if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/") {
		logNonOK(403)
		http.Error(w, dirListNotAllowed, 403)
		return
	}

	cw := staticFileResponseWriter{w, http.StatusOK}
	staticFS.ServeHTTP(&cw, r)
	if cw.statusCode >= 400 {
		logNonOK(cw.statusCode)
	}
}

type staticFileResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *staticFileResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.Header().Set(strCacheControlHdr, "max-age=31536000, immutable")
	lrw.ResponseWriter.WriteHeader(code)
}
