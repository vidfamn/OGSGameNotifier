package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vidfamn/OGSGameNotifier/internal/api"
	"github.com/vidfamn/OGSGameNotifier/internal/websocket"
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

		authResponse, err := api.PostAuthorize(username, password, OAuthClientID, OAuthClientSecret)
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

	auth := &api.PostAuthorizeResponse{}
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

	if *ws {
		if err := websocket.OGSWebSocket(auth.AccessToken); err != nil {
			logrus.Error(err)
			return
		}
	}

	// players, err := api.GetPlayers()
	// if err != nil {
	// 	logrus.Error(err)
	// 	return
	// }

	// db, err := memdb.NewMemDB(storage.Schema())
	// if err != nil {
	// 	logrus.Error(err)
	// 	return
	// }

	// txn := db.Txn(true)
	// for _, p := range players {
	// 	if err := txn.Insert("player", p); err != nil {
	// 		logrus.WithFields(logrus.Fields{
	// 			"player": p,
	// 			"error":  err,
	// 		}).Warn("could not store player")
	// 		continue
	// 	} else {
	// 		logrus.WithFields(logrus.Fields{
	// 			"player": p,
	// 		}).Debug("stored player in memdb")
	// 	}
	// }
	// txn.Commit()

	// txn = db.Txn(false)
	// it, err := txn.Get("player", "ratings.overall.rating")
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println("All the players:")
	// for obj := it.Next(); obj != nil; obj = it.Next() {
	// 	p := obj.(*api.Player)
	// 	logrus.WithFields(logrus.Fields{
	// 		"ID":       p.ID,
	// 		"Username": p.Username,
	// 		"Rating":   p.Ratings.Overall.Rating,
	// 	}).Info("found player")
	// }

	// _, err = getChallenges()
	// if err != nil {
	// 	logrus.Error(err)
	// 	return
	// }
}
