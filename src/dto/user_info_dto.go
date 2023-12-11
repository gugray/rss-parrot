package dto

type UserInfo struct {
	Context           []string  `json:"@context"`
	Id                string    `json:"id"`
	Type              string    `json:"type"`
	PreferredUserName string    `json:"preferredUsername"`
	Inbox             string    `json:"inbox"`
	Outbox            string    `json:"outbox"`
	PublicKey         PublicKey `json:"publicKey"`
}

type PublicKey struct {
	Id           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}
