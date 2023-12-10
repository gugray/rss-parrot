package internal

import (
	"io"
	"log"
	"net/http"
)

type EchoHandler struct{}

func NewEchoHandler() *EchoHandler {
	return &EchoHandler{}
}

func (*EchoHandler) Pattern() string {
	return "/echo"
}

func (*EchoHandler) Method() string {
	return "GET"
}

func (*EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(w, r.Body); err != nil {
		log.Printf("Failed to handle request: %v\n", err)
	}
}
