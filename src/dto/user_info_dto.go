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
}

type UserEndpoints struct {
	SharedInbox string `json:"sharedInbox"`
}

type PublicKey struct {
	Id           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}
