package dto

type Note struct {
	Id           string   `json:"id"`
	Type         string   `json:"type"`
	Published    string   `json:"published"`
	Summary      *string  `json:"summary"`
	AttributedTo string   `json:"attributedTo"`
	InReplyTo    *string  `json:"inReplyTo"`
	Content      string   `json:"content"`
	To           []string `json:"to"`
	Cc           []string `json:"cc"`
}
