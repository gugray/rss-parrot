package dto

type ActivityInBase struct {
	Id    string `json:"id"`
	Type  string `json:"type"`
	Actor string `json:"actor"`
}

type ActivityInStringObject struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	Actor  string `json:"actor"`
	Object string `json:"object"`
}

type ActivityOut struct {
	Context any    `json:"@context"`
	Id      string `json:"id"`
	Type    string `json:"type"`
	Actor   string `json:"actor"`
	Object  any    `json:"object,omitempty"`
}
