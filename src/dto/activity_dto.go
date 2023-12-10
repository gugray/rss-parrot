package dto

type Activity struct {
	Context string `json:"@context"`
	Id      string `json:"id"`
	Type    string `json:"type"`
	Actor   string `json:"actor"`
	Object  any    `json:"object,omitempty"`
}

type Note struct {
	Id           string   `json:"id"`
	Type         string   `json:"type"`
	Published    string   `json:"published"`
	AttributedTo string   `json:"attributedTo"`
	InReplyTo    *string  `json:"inReplyTo"`
	Content      string   `json:"content"`
	To           []string `json:"to"`
	Cc           []string `json:"cc"`
}
