package server

import (
	"context"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"rss_parrot/logic"
	"rss_parrot/shared"
	"strconv"
	"strings"
	"sync/atomic"
)

const assetsDir = "/assets"
const faviconName = "/favicon.ico"
const chunkSize = 65536
const strCacheControlHdr = "Cache-Control"

func NewHTTPServer(cfg *shared.Config,
	logger shared.ILogger,
	lc fx.Lifecycle,
	router *mux.Router,
	metrics logic.IMetrics,
) *http.Server {
	addStr := ":" + strconv.FormatUint(uint64(cfg.ServicePort), 10)
	srv := &http.Server{Addr: addStr, Handler: trimSlashHandler(router)}
	srv.ConnState = func(conn net.Conn, state http.ConnState) {
		handleConnState(metrics, conn, state)
	}
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

var connCount int32

func handleConnState(metrics logic.IMetrics, conn net.Conn, state http.ConnState) {
	var newCount int32 = -1
	if state == http.StateNew {
		newCount = atomic.AddInt32(&connCount, 1)
	} else if state == http.StateClosed {
		newCount = atomic.AddInt32(&connCount, -1)
	}
	if newCount != -1 {
		metrics.CurrentConnections(int(newCount))
	}
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

	var notFoundHandler func(w http.ResponseWriter, r *http.Request) = nil

	router := mux.NewRouter()
	for _, group := range groups {
		subRouter := router.PathPrefix(group.Prefix()).Subrouter()
		authMW := group.AuthMW()
		subRouter.Use(noCacheMW)
		subRouter.Use(authMW)
		for _, def := range group.GroupDefs() {
			if def.pattern == rootPlacholder {
				router.HandleFunc("/", def.handler).Methods("OPTIONS", def.method)
			} else if def.pattern == notFoundPlacholder {
				notFoundHandler = def.handler
			} else {
				subRouter.HandleFunc(def.pattern, def.handler).Methods("OPTIONS", def.method)
			}
		}
	}
	// Static files with error logging
	// HEAD requests: 405
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			handleStatic(logger, notFoundHandler, w, r)
		} else if r.Method == "HEAD" {
			logger.Infof("Rejecting HEAD: %s", r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
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

func handleStatic(logger shared.ILogger,
	notFoundHandler func(w http.ResponseWriter, r *http.Request),
	w http.ResponseWriter, r *http.Request,
) {

	return404 := func() {
		logger.Infof("%s %s returns 404", r.Method, r.URL.Path)
		if notFoundHandler == nil {
			http.Error(w, notFoundStr, http.StatusNotFound)
		} else {
			notFoundHandler(w, r)
		}
	}

	w.Header().Set(strCacheControlHdr, "max-age=31536000, immutable")

	// We serve everything from /assets folder, EXCEPT favicon.ico, which gets special treatment
	if r.URL.Path != faviconName && !strings.HasPrefix(r.URL.Path, assetsDir) {
		return404()
		return
	}

	fn := filepath.Join(wwwPathPrefx, r.URL.Path)
	if r.URL.Path == faviconName {
		fn = filepath.Join(wwwPathPrefx, assetsDir, r.URL.Path)
	}
	file, err := os.Open(fn)
	if err != nil {
		return404()
		return
	}
	defer file.Close()

	var fi os.FileInfo
	fi, err = file.Stat()
	if err != nil {
		return404()
		return
	}

	headersSent := false
	writeHeaders := func() {
		w.Header().Set("Content-Length", strconv.Itoa(int(fi.Size())))
		w.Header().Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
		if strings.HasSuffix(r.URL.Path, ".svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		} else if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
	}

	buf := make([]byte, chunkSize)
	for {
		n, err := file.Read(buf)
		if err != nil {
			logger.Errorf("Error reading file %s: %v", fn, err)
			return404()
			return
		}
		if !headersSent {
			writeHeaders()
			headersSent = true
		}
		if n > 0 {
			_, err := w.Write(buf[0:n])
			if err != nil {
				logger.Errorf("Error writing response: %v", err)
				return
			}
		}
		if n < chunkSize {
			break
		}
	}
}
