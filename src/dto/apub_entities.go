package dto

type UserInfo struct {
	Context           any           `json:"@context"`
	Id                string        `json:"id"`
	Type              string        `json:"type"`
	PreferredUserName string        `json:"preferredUsername"`
	Name              string        `json:"name"`
	Summary           string        `json:"summary"`
	ManuallyApproves  bool          `json:"manuallyApprovesFollowers"`
	Published         string        `json:"published"`
	Inbox             string        `json:"inbox"`
	Outbox            string        `json:"outbox"`
	Followers         string        `json:"followers"`
	Following         string        `json:"following"`
	Endpoints         UserEndpoints `json:"endpoints"`
	PublicKey         PublicKey     `json:"publicKey"`
	Attachments       []Attachment  `json:"attachment"`
	Icon              Image         `json:"icon"`
	Image             Image         `json:"image"`
}

type Attachment struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Image struct {
	Type string `json:"type"`
	Url  string `json:"url"`
}

type UserEndpoints struct {
	SharedInbox string `json:"sharedInbox"`
}

type PublicKey struct {
	Id           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

type OrderedListSummary struct {
	Context    any     `json:"@context"`
	Id         string  `json:"id"`
	Type       string  `json:"type"`
	TotalItems uint    `json:"totalItems"`
	First      *string `json:"first,omitempty"`
	Last       *string `json:"last,omitempty"`
}

type ActivityInBase struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	Actor  string `json:"actor"`
	Object any    `json:"object"`
}

type ActivityIn[T any] struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	Actor  string `json:"actor"`
	Object T      `json:"object"`
}

type ActivityOut struct {
	Context any    `json:"@context"`
	Id      string `json:"id"`
	Type    string `json:"type"`
	Actor   string `json:"actor"`
	Object  any    `json:"object,omitempty"`
}

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
