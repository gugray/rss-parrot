package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const (
	internalErrorStr  = "Internal Server Error"
	invalidRequestStr = "Invalid Request"
	notFoundStr       = "Not Found"
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
func writeResponse(w http.ResponseWriter, resp interface{}) {
	var err error
	var respJson []byte
	if respJson, err = json.Marshal(resp); err != nil {
		log.Printf("Failed to serialize response: %v\n", err)
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}
	if _, err = fmt.Fprintln(w, string(respJson)); err != nil {
		log.Printf("Failed to write response: %v\n", err)
		http.Error(w, internalErrorStr, http.StatusInternalServerError)
		return
	}
}
