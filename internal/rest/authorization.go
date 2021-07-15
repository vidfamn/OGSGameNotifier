package rest

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
)

type PostAuthorizeResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"Bearer"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

func PostAuthorize(username, password, clientID, clientSecret string) (*PostAuthorizeResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "password")
	data.Set("username", username)
	data.Set("password", password)

	req, _ := http.NewRequest(http.MethodPost, "https://online-go.com/oauth2/token/", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	logrus.WithFields(logrus.Fields{
		"body":    data,
		"headers": req.Header,
	}).Debug("POST token request")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, errors.New("error sending request: " + err.Error())
	}

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		logrus.WithFields(logrus.Fields{
			"status": resp.Status,
			"body":   string(respBody),
		}).Debug("POST token response")
		return nil, errors.New("unexpected response status")
	}

	response := &PostAuthorizeResponse{}
	if err := json.Unmarshal(respBody, response); err != nil {
		logrus.WithFields(logrus.Fields{
			"status": resp.Status,
			"body":   string(respBody),
		}).Debug("POST token response")
		return nil, errors.New("could not decode json")
	}

	logrus.WithFields(logrus.Fields{
		"status":   resp.Status,
		"response": response,
	}).Debug("POST token response")

	return response, nil
}
