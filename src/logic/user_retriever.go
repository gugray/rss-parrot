package logic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"rss_parrot/dto"
	"rss_parrot/shared"
)

type IUserRetriever interface {
	Retrieve(userUrl string) (info *dto.UserInfo, err error)
}

type userRetriever struct {
	cfg *shared.Config
}

func NewUserRetriever(cfg *shared.Config) IUserRetriever {
	return &userRetriever{cfg}
}

func (ur *userRetriever) Retrieve(userUrl string) (info *dto.UserInfo, err error) {

	client := &http.Client{}
	var req *http.Request
	if req, err = http.NewRequest("GET", userUrl, nil); err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user profile; got status %v", resp.StatusCode)
	}

	defer resp.Body.Close()
	var bodyBytes []byte
	if bodyBytes, err = io.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	var obj dto.UserInfo
	if err = json.Unmarshal(bodyBytes, &obj); err != nil {
		return nil, err
	}

	return &obj, nil
}
