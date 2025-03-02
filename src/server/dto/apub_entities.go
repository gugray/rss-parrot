package dto

import (
	"encoding/json"
	"errors"
	"fmt"
)

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

func getRecipient(raw any) ([]string, error) {
	var res []string
	if raw == nil {
		return res, nil
	}
	if slice, ok := raw.([]interface{}); ok {
		for _, s := range slice {
			if str, ok := s.(string); ok {
				res = append(res, str)
			} else {
				return res, fmt.Errorf("list of recipients must only contain strings")
			}
		}
	} else if str, ok := raw.(string); ok {
		res = []string{str}
	} else {
		return res, fmt.Errorf("to and cc must be single string or array of strings")
	}
	return res, nil
}

type ActivityInBase struct {
	Id     string   `json:"id"`
	Type   string   `json:"type"`
	Actor  string   `json:"actor"`
	To     []string `json:"-"`
	RawTo  any      `json:"to"`
	Cc     []string `json:"-"`
	RawCc  any      `json:"cc"`
	Object any      `json:"object"`
}

func (x *ActivityInBase) UnmarshalJSON(data []byte) error {
	var err error
	type Y ActivityInBase
	var y = (*Y)(x)
	if err = json.Unmarshal(data, y); err != nil {
		return err
	}
	if y.To, err = getRecipient(y.RawTo); err != nil {
		return err
	}
	if y.Cc, err = getRecipient(y.RawCc); err != nil {
		return err
	}
	return nil
}

type ActivityIn[T any] struct {
	Id     string   `json:"id"`
	Type   string   `json:"type"`
	Actor  string   `json:"actor"`
	To     []string `json:"-"`
	RawTo  any      `json:"to"`
	Cc     []string `json:"-"`
	RawCc  any      `json:"cc"`
	Object T        `json:"object"`
}

func (x *ActivityIn[T]) UnmarshalJSON(data []byte) error {
	var err error
	type Y ActivityIn[T]
	var y = (*Y)(x)
	if err = json.Unmarshal(data, y); err != nil {
		return err
	}
	if y.To, err = getRecipient(y.RawTo); err != nil {
		return err
	}
	if y.Cc, err = getRecipient(y.RawCc); err != nil {
		return err
	}
	return nil
}

type ActivityOut struct {
	Context any       `json:"@context"`
	Id      string    `json:"id"`
	Type    string    `json:"type"`
	Actor   string    `json:"actor"`
	To      *[]string `json:"to,omitempty"`
	Cc      *[]string `json:"cc,omitempty"`
	Object  any       `json:"object,omitempty"`
}

type Note struct {
	Context      string   `json:"@context,omitempty"`
	Id           string   `json:"id"`
	Type         string   `json:"type"`
	Published    string   `json:"published"`
	Summary      *string  `json:"summary"`
	AttributedTo string   `json:"attributedTo"`
	InReplyTo    *string  `json:"inReplyTo"`
	To           []string `json:"-"`
	RawTo        any      `json:"to"`
	Cc           []string `json:"-"`
	RawCc        any      `json:"cc"`
	Content      string   `json:"content"`
	Tag          *[]Tag   `json:"-"`
	RawTag       any      `json:"tag,omitempty"`
}

func (x *Note) UnmarshalJSON(data []byte) error {
	var err error
	type Y Note
	var y = (*Y)(x)
	if err = json.Unmarshal(data, y); err != nil {
		return err
	}
	if y.To, err = getRecipient(y.RawTo); err != nil {
		return err
	}
	if y.Cc, err = getRecipient(y.RawCc); err != nil {
		return err
	}
	if y.Tag, err = getTag(y.RawTag); err != nil {
		return err
	}
	return nil
}

func (x *Note) MarshalJSON() ([]byte, error) {
	type Y Note
	var y = (*Y)(x)
	y.RawTo = y.To
	y.RawCc = y.Cc
	y.RawTag = y.Tag
	return json.Marshal(y)
}

type Tag struct {
	Type string `json:"type"`
	Href string `json:"href"`
	Name string `json:"name"`
}

func getTag(raw any) (*[]Tag, error) {
	// No value is legit
	if raw == nil {
		return nil, nil
	}

	retrieve := func(obj *map[string]interface{}) (*Tag, error) {
		var tag Tag
		var ok bool
		if tag.Href, ok = (*obj)["href"].(string); !ok {
			return nil, errors.New("invalid data in tag's 'href' property; string expected")
		}
		if tag.Name, ok = (*obj)["name"].(string); !ok {
			return nil, errors.New("invalid data in tag's 'name' property; string expected")
		}
		if tag.Type, ok = (*obj)["type"].(string); !ok {
			return nil, errors.New("invalid data in tag's 'type' property; string expected")
		}
		return &tag, nil
	}

	// Single Tag object
	if obj, ok := raw.(map[string]interface{}); ok {
		if tag, err := retrieve(&obj); err != nil {
			return nil, err
		} else {
			return &[]Tag{*tag}, nil
		}
	}
	// Array
	if slice, ok := raw.([]interface{}); ok {
		var res []Tag
		for _, s := range slice {
			if obj, ok := s.(map[string]interface{}); ok {
				if tag, err := retrieve(&obj); err != nil {
					return nil, err
				} else {
					res = append(res, *tag)
				}
			} else {
				return nil, errors.New("unexpected item in 'tag' array; must only contain tag objects")
			}
		}
		return &res, nil
	}
	return nil, errors.New("invalid data in 'tag' property")
}
