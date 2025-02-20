package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"rss_parrot/dal"
	"rss_parrot/dto"
	"rss_parrot/logic"
	"rss_parrot/shared"
)

// curl -X POST -H "X-API-KEY: 5QLbv8hrifgdXCEN" 'https://rss-parrot.zydeo.net/api/actions/vacuum'
// curl -X POST -H "X-API-KEY: 5QLbv8hrifgdXCEN" 'https://rss-parrot.zydeo.net/api/actions/pprof'

type apiHandlerGroup struct {
	cfg    *shared.Config
	logger shared.ILogger
	fdfol  logic.IFeedFollower
	repo   dal.IRepo
	prof   logic.IProfiler
}

func NewApiHandlerGroup(
	cfg *shared.Config,
	logger shared.ILogger,
	fdfol logic.IFeedFollower,
	repo dal.IRepo,
	prof logic.IProfiler,
) IHandlerGroup {
	res := apiHandlerGroup{
		cfg:    cfg,
		logger: logger,
		fdfol:  fdfol,
		repo:   repo,
		prof:   prof,
	}
	return &res
}

func (hg *apiHandlerGroup) Prefix() string {
	return "/api"
}

func (hg *apiHandlerGroup) GroupDefs() []handlerDef {
	return []handlerDef{
		{"POST", "/feeds", func(w http.ResponseWriter, r *http.Request) { hg.postFeeds(w, r) }},
		{"DELETE", "/accounts/{account}", func(w http.ResponseWriter, r *http.Request) { hg.deleteAccount(w, r) }},
		{"POST", "/actions/vacuum", func(w http.ResponseWriter, r *http.Request) { hg.postActionsVacuum(w, r) }},
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

func (hg *apiHandlerGroup) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var err error
	hg.logger.Infof("Handling %s %s", r.Method, r.URL.Path)

	accountName := mux.Vars(r)["account"]
	if accountName == "" {
		msg := "Missing account parameter"
		hg.logger.Info(msg)
		writeErrorResponse(w, msg, http.StatusBadRequest)
	}

	var acct *dal.Account
	acct, err = hg.repo.GetAccount(accountName)
	if err != nil {
		msg := fmt.Sprintf("Failed to get account: %v", err)
		hg.logger.Error(msg)
		writeErrorResponse(w, msg, http.StatusInternalServerError)
		return
	}
	if acct == nil {
		msg := fmt.Sprintf("Account not found: %s", accountName)
		writeErrorResponse(w, msg, http.StatusNotFound)
		return
	}

	err = hg.repo.BruteDeleteAccount(acct.Id)
	if err != nil {
		msg := fmt.Sprintf("Failed to brute-delete account: %v", err)
		hg.logger.Error(msg)
		writeErrorResponse(w, msg, http.StatusInternalServerError)
		return
	}

	writeJsonResponse(hg.logger, w, rtPlainJson, "OK")
}

func (hg *apiHandlerGroup) postActionsVacuum(w http.ResponseWriter, r *http.Request) {
	hg.logger.Infof("Handling %s %s", r.Method, r.URL.Path)

	go func() {
		if err := hg.repo.Vacuum(); err != nil {
			msg := fmt.Sprintf("Error vacuuming DB: %v", err)
			hg.logger.Error(msg)
			writeErrorResponse(w, msg, http.StatusBadRequest)
			return
		}
		hg.logger.Info("Finished vacuuming successfully")
	}()

	writeJsonResponse(hg.logger, w, rtPlainJson, "OK")
}

func (hg *apiHandlerGroup) postFeeds(w http.ResponseWriter, r *http.Request) {
	var err error
	hg.logger.Infof("Handling %s %s", r.Method, r.URL.Path)

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
	writeJsonResponse(hg.logger, w, rtPlainJson, res)
}
