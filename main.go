package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/vidfamn/OGSGameNotifier/internal/rest"
	"github.com/vidfamn/OGSGameNotifier/internal/storage"
	"github.com/vidfamn/OGSGameNotifier/internal/websocket"

	"github.com/sirupsen/logrus"
)

var (
	Application string = "OGSGameNotifier"

	// Overridden at compile time on make build
	Version           string = "dev"
	OAuthClientID     string = "dev"
	OAuthClientSecret string = "dev"

	authorizationFile string = ".authorization"
)

func main() {
	version := flag.Bool("version", false, "prints application version")
	debug := flag.Bool("debug", false, "debug log")
	authorize := flag.Bool("authorize", false, "creates and stores authorization to be used by the application")
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

	if *authorize {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("username: ")
		username, err := reader.ReadString('\n')
		if err != nil {
			logrus.WithField("error", err).Error("could not read input")
			return
		}
		username = strings.Replace(username, "\n", "", -1)

		fmt.Print("password: ")
		password, err := reader.ReadString('\n')
		if err != nil {
			logrus.WithField("error", err).Error("could not read input")
			return
		}
		password = strings.Replace(password, "\n", "", -1)

		authResponse, err := rest.PostAuthorize(username, password, OAuthClientID, OAuthClientSecret)
		if err != nil {
			logrus.Error(err)
			return
		}

		b, _ := json.Marshal(authResponse)

		if err := ioutil.WriteFile(authorizationFile, b, 0644); err != nil {
			logrus.WithFields(logrus.Fields{
				"authorization":      authResponse,
				"authorization_file": authorizationFile,
			}).Error("could not write file")
			return
		}

		logrus.WithFields(logrus.Fields{
			"authorization":      authResponse,
			"authorization_file": authorizationFile,
		}).Info("created and stored authorization token in file")
		return
	}

	b, err := ioutil.ReadFile(authorizationFile)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"authorization_file": authorizationFile,
			"error":              err,
		}).Error("could not read file, see --help")
		return
	}

	auth := &rest.PostAuthorizeResponse{}
	if err := json.Unmarshal(b, auth); err != nil {
		logrus.WithFields(logrus.Fields{
			"authorization_file": authorizationFile,
			"error":              err,
		}).Error("could not unmarshal json")
		return
	}

	logrus.WithFields(logrus.Fields{
		"authorization": auth,
	}).Debug("found stored .authorization file")

	client, err := websocket.NewOGSWebSocket(auth.AccessToken)
	if err != nil {
		logrus.Error(err)
		return
	}

	gameListResponse, err := client.GameListRequest(&websocket.GameListQueryRequest{
		List:   "live",
		SortBy: "rank",
		Where:  map[string]interface{}{},
		From:   0,
		Limit:  100,
	}, time.Second*30)
	if err != nil {
		logrus.Error(err)
		return
	}

	db, err := memdb.NewMemDB(storage.Schema())
	if err != nil {
		logrus.Error(err)
		return
	}

	txn := db.Txn(true)
	for _, game := range gameListResponse.Results {
		if err := txn.Insert("game", game); err != nil {
			logrus.WithFields(logrus.Fields{
				"game_id": game.ID,
				"error":   err,
			}).Warn("could not store in memdb")
			continue
		} else {
			logrus.WithFields(logrus.Fields{
				"game":         game.ID,
				"white_rating": game.White.Ratings.Overall.Rating,
				"black_rating": game.Black.Ratings.Overall.Rating,
			}).Debug("stored in memdb")
		}
	}
	txn.Commit()

	txn = db.Txn(false)
	it, err := txn.LowerBound("game", "white.ratings.overall.rating", float64(2000))
	if err != nil {
		panic(err)
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		game := obj.(*websocket.Game)
		logrus.WithFields(logrus.Fields{
			"id":           game.ID,
			"white_rating": game.White.Ratings.Overall.Rating,
			"white_name":   game.White.Username,
		}).Info("found player")
	}

	it, err = txn.LowerBound("game", "black.ratings.overall.rating", float64(2000))
	if err != nil {
		panic(err)
	}
	for obj := it.Next(); obj != nil; obj = it.Next() {
		game := obj.(*websocket.Game)
		logrus.WithFields(logrus.Fields{
			"id":           game.ID,
			"black_rating": game.Black.Ratings.Overall.Rating,
			"black_name":   game.Black.Username,
		}).Info("found player")
	}

	var stopChan = make(chan os.Signal, 2)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-stopChan // wait for SIGINT
}
