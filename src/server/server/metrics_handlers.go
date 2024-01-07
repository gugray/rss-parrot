package server

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"rss_parrot/shared"
	"strings"
)

type metricsHandlerGroup struct {
	cfg             *shared.Config
	logger          shared.ILogger
	promHttpHandler http.Handler
}

func NewMetricsHandlerGroup(
	cfg *shared.Config,
	logger shared.ILogger,
) IHandlerGroup {
	res := metricsHandlerGroup{
		cfg:             cfg,
		logger:          logger,
		promHttpHandler: promhttp.Handler(),
	}
	return &res
}

func (hg *metricsHandlerGroup) Prefix() string {
	return "/"
}

func (hg *metricsHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"GET", "/metrics", func(w http.ResponseWriter, r *http.Request) { hg.getMetrics(w, r) }},
	}
}

func (hg *metricsHandlerGroup) AuthMW() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return hg.authMW(next)
	}
}

func (hg *metricsHandlerGroup) authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authSecret := ""
		var authHeader = r.Header.Get(metricsAuthHeader)
		if strings.HasPrefix(authHeader, "Bearer ") {
			authSecret = strings.TrimPrefix(authHeader, "Bearer ")
		}
		if authSecret == "" || authSecret != hg.cfg.Secrets.MetricsAuth {
			hg.logger.Warnf("Metrics scrape request with missing or invalid Authorization header '%s' s", authSecret)
			writeErrorResponse(w, badAuthorization, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (hg *metricsHandlerGroup) getMetrics(w http.ResponseWriter, r *http.Request) {
	hg.logger.Infof("Handling metrics GET: %s", r.URL.Path)
	hg.promHttpHandler.ServeHTTP(w, r)
}
