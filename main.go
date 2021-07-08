package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	Application       string = "OGSGameNotifier"
	Version           string = "dev"
	OAuthClientID            = "dev"
	OAuthClientSecret        = "dev"
)

func main() {
	version := flag.Bool("version", false, "prints application version")
	debug := flag.Bool("debug", false, "debug log")
	auth := flag.Bool("authorize", false, "authorizes and creates a token to be used by the application")
	flag.Parse()

	logrus.SetFormatter(&logrus.TextFormatter{})

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if *version {
		logrus.WithFields(logrus.Fields{
			Application: Application,
			Version:     Version,
		}).Info("application info")
		return
	}

	if *auth {
		token, err := postAuthorize()
		if err != nil {
			logrus.Error(err)
			return
		}

		filename := ".token"
		logrus.WithFields(logrus.Fields{
			"token": token,
			"file":  filename,
		}).Info("created and stored authorization token in file")
		return
	}

	_, err := getChallenges()
	if err != nil {
		logrus.Error(err)
		return
	}
}

func getChallenges() (string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://online-go.com/api/v1/challenges/", nil)
	logrus.Debug("GET challenges request")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.New("could not get challenges: " + err.Error())
	}

	respBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected response, status: %s, body: %s", resp.Status, respBody)
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.Status,
		"body":   string(respBody),
	}).Debug("GET challenges response")

	return "", nil
}

type tokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

func postAuthorize() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.New("could not read input: " + err.Error())
	}
	username = strings.Replace(username, "\n", "", -1)

	fmt.Print("password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.New("could not read input: " + err.Error())
	}
	password = strings.Replace(password, "\n", "", -1)

	tokenRequest := &tokenRequest{
		ClientID:     OAuthClientID,
		ClientSecret: OAuthClientSecret,
		GrantType:    "password",
		Username:     username,
		Password:     password,
	}
	b, err := json.Marshal(tokenRequest)
	if err != nil {
		return "", errors.New("could not marshal request: " + err.Error())
	}

	req, _ := http.NewRequest(http.MethodPost, "https://online-go.com/oauth2/token/", bytes.NewBuffer(b))
	req.Header.Add("Content-Type", "application/json")

	logrus.WithFields(logrus.Fields{
		"body":    string(b),
		"headers": req.Header,
	}).Debug("POST token request")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return "", errors.New("error sending request: " + err.Error())
	}

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected response, status: %s, body: %s", resp.Status, respBody)
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.Status,
		"body":   string(respBody),
	}).Debug("POST token response")

	return "", nil
}
