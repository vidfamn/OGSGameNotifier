package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	ws := flag.Bool("websocket", false, "use streaming websocket to fetch games")
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

	if *ws {
		// if err := websocket.OGSWebSocket(); err != nil {
		// 	logrus.Error(err)
		// 	return
		// }
	}

	_, err := getChallenges()
	if err != nil {
		logrus.Error(err)
		return
	}
}

func socketGetChallenges() (string, error) {

	// 'https://online-go.com/socket.io', transports='websocket'
	return "", nil
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

	data := url.Values{}
	data.Set("client_id", OAuthClientID)
	data.Set("client_secret", OAuthClientSecret)
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
