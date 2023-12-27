package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"rss_parrot/shared"
)

const (
	internalErrorStr = "Internal Server Error"
	badRequestStr    = "Invalid Request"
	notFoundStr      = "Not Found"
)

// Defines a single HTTP handler (endpoint)
type handlerDef struct {
	method  string
	pattern string
	handler func(http.ResponseWriter, *http.Request)
}

// IHandlerGroup groups together multiple HTTP handler definitions.
type IHandlerGroup interface {
	GroupDefs() []handlerDef
}

// Returns the JSON serialized object as the response body; handles errors.
func writeJsonResponse(logger shared.ILogger, w http.ResponseWriter, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	var err error
	var respJson []byte
	if respJson, err = json.Marshal(resp); err != nil {
		logger.Warnf("Failed to serialize response: %v\n", err)
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}
	if _, err = fmt.Fprintln(w, string(respJson)); err != nil {
		logger.Warnf("Failed to write response: %v\n", err)
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}
}

type errorResp struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

func writeErrorResponse(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	resp := errorResp{msg, code}
	respJson, _ := json.Marshal(resp)
	http.Error(w, string(respJson), code)
}

func readBody(logger shared.ILogger, w http.ResponseWriter, r *http.Request) []byte {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Warnf("Failed to read request body: %v", err)
		http.Error(w, badRequestStr, http.StatusBadRequest)
		return nil
	}
	return body
}
