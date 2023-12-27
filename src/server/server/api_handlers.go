package server

import (
	"net/http"
	"rss_parrot/logic"
	"rss_parrot/shared"
)

type apiHandlerGroup struct {
	cfg         *shared.Config
	logger      shared.ILogger
	keyStore    logic.IKeyStore
	sender      logic.IActivitySender
	broadcaster logic.IMessenger
}

func NewApiHandlerGroup(
	cfg *shared.Config,
	logger shared.ILogger,
	keyStore logic.IKeyStore,
	sender logic.IActivitySender,
	broadcaster logic.IMessenger,
) IHandlerGroup {
	res := apiHandlerGroup{
		cfg:         cfg,
		logger:      logger,
		keyStore:    keyStore,
		sender:      sender,
		broadcaster: broadcaster,
	}
	return &res
}

func (hg *apiHandlerGroup) Prefix() string {
	return "/api"
}

func (hg *apiHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"POST", "/feeds", func(w http.ResponseWriter, r *http.Request) { hg.postFeeds(w, r) }},
	}
}

func (hg *apiHandlerGroup) AuthMW() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return hg.authMW(next)
	}
}

func (hg *apiHandlerGroup) authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var apiKey = r.Header.Get(apiKeyHeader)
		found := false
		for _, key := range hg.cfg.Secrets.ApiKeys {
			if apiKey == key {
				found = true
			}
		}
		if !found {
			keyPart := apiKey
			if len(apiKey) > 4 {
				keyPart = apiKey[:4] + "..."
			}
			hg.logger.Warnf("API request with missing or invalid key '%s': %s", keyPart, r.URL.Path)
			writeErrorResponse(w, badApiKeyStr, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (hg *apiHandlerGroup) postFeeds(w http.ResponseWriter, r *http.Request) {
	hg.logger.Info("POST /api/feeds: Request received")
}
