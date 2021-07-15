package rest

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

type Player struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Country  string `json:"country"`
	Icon     string `json:"icon"`
	Ratings  struct {
		Version int64 `json:"version"`
		Overall struct {
			Rating     float64 `json:"rating"`
			Deviation  float64 `json:"deviation"`
			Volatility float32 `json:"volatility"`
		} `json:"overall"`
	} `json:"ratings"`
}

type playersResponse struct {
	Count    int       `json:"count"`
	Next     string    `json:"next"`
	Previous string    `json:"previous"`
	Results  []*Player `json:"results"`
}

func GetPlayers() ([]*Player, error) {
	url := "https://online-go.com/api/v1/players"

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, errors.New("could not get players: " + err.Error())
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("could not read response: " + err.Error())
	}

	response := &playersResponse{}
	if err := json.Unmarshal(respBody, response); err != nil {
		return nil, errors.New("could not unmarshal: " + err.Error())
	}

	return response.Results, nil
}
