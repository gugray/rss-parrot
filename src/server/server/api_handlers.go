package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"rss_parrot/dto"
	"rss_parrot/logic"
	"rss_parrot/shared"
)

type apiHandlerGroup struct {
	cfg    *shared.Config
	logger shared.ILogger
	fdfol  logic.IFeedFollower
}

func NewApiHandlerGroup(
	cfg *shared.Config,
	logger shared.ILogger,
	fdfol logic.IFeedFollower,
) IHandlerGroup {
	res := apiHandlerGroup{
		cfg:    cfg,
		logger: logger,
		fdfol:  fdfol,
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

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			return
		}

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
	var err error
	hg.logger.Info("POST /api/feeds: Request received")

	// Read and parse body
	bodyBytes := readBody(hg.logger, w, r)
	if bodyBytes == nil {
		hg.logger.Info("Empty request body")
		writeErrorResponse(w, "Request body must not be empty", http.StatusBadRequest)
		return
	}
	var feed dto.Feed
	if err = json.Unmarshal(bodyBytes, &feed); err != nil {
		msg := fmt.Sprintf("Invalid JSON in request body: %v", err)
		hg.logger.Info(msg)
		writeErrorResponse(w, msg, http.StatusBadRequest)
		return
	}

	acct, status, feedErr := hg.fdfol.GetAccountForFeed(feed.SiteUrl)
	if feedErr != nil {
		msg := fmt.Sprintf("Failed to get feed: %v", feedErr)
		writeErrorResponse(w, msg, http.StatusInternalServerError)
		return
	}
	if status < 0 {
		msg := fmt.Sprintf("Feed is banned: %d", status)
		writeErrorResponse(w, msg, http.StatusInternalServerError)
		return
	}
	res := dto.Feed{
		CreatedAt:       acct.CreatedAt,
		UserUrl:         acct.UserUrl,
		Handle:          acct.Handle,
		FeedName:        acct.FeedName,
		FeedSummary:     acct.FeedSummary,
		ProfileImageUrl: acct.ProfileImageUrl,
		SiteUrl:         acct.SiteUrl,
		FeedUrl:         acct.FeedUrl,
		FeedLastUpdated: acct.FeedLastUpdated,
		NextCheckDue:    acct.NextCheckDue,
	}

	if status == logic.FsNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	writeJsonResponse(hg.logger, w, res)
}
