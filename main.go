package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-memdb"
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

type Notifier struct {
	OGS *websocket.OGSWebSocket
	DB  *memdb.MemDB

	BotGames        bool
	MinMedianRating float64
	BoardSize       int

	NotifyGames map[int64]*websocket.Game
}

func main() {
	version := flag.Bool("version", false, "prints application version")
	logLevel := flag.String("log-level", "warn", "log level: error, warn, info, debug")
	flag.Parse()

	logrus.SetFormatter(&logrus.TextFormatter{})

	switch *logLevel {
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	default:
		logrus.SetLevel(logrus.WarnLevel)
	}

	if *version {
		logrus.WithFields(logrus.Fields{
			Application: Application,
			Version:     Version,
		}).Info("application info")
		return
	}

	ogs, err := websocket.NewOGSWebSocket()
	if err != nil {
		logrus.Error(err)
		return
	}

	db, err := memdb.NewMemDB(storage.Schema())
	if err != nil {
		logrus.Error(err)
		return
	}

	notifier := &Notifier{
		OGS:             ogs,
		DB:              db,
		BotGames:        false,
		MinMedianRating: 2100,
		BoardSize:       19,
		NotifyGames:     map[int64]*websocket.Game{},
	}

	var stopChan = make(chan os.Signal, 2)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	pollingTicker := time.NewTicker(time.Second * 30)

	notifier.updateGameList()

	for {
		select {
		case <-pollingTicker.C:
			notifier.updateGameList()

			// Notify new games
			for _, game := range notifier.NotifyGames {
				logrus.WithFields(logrus.Fields{
					"id":   game.ID,
					"game": gameStr(game),
				}).Info("would notify game")
			}

			// Notifications sent, clear the list
			notifier.NotifyGames = map[int64]*websocket.Game{}

			// All games
			txn := notifier.DB.Txn(false)
			it, err := txn.Get("games", "id")
			if err != nil {
				logrus.WithError(err).Error("could not get games from memdb")
				continue
			}

			for obj := it.Next(); obj != nil; obj = it.Next() {
				game, ok := obj.(*websocket.Game)
				if !ok {
					logrus.Error("expected *websocket.Game")
					continue
				}

				logrus.WithFields(logrus.Fields{
					"id":   game.ID,
					"game": gameStr(game),
				}).Info("found matching game")
			}

		case <-stopChan:
			pollingTicker.Stop()
			return
		}
	}
}

func (n *Notifier) updateGameList() {
	gameListResponse, err := n.OGS.GameListRequest(&websocket.GameListQueryRequest{
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

	txn := n.DB.Txn(true)
	txn.TrackChanges()

	deleted, err := txn.DeleteAll("games", "id")
	if err != nil {
		logrus.WithError(err).Error("could not delete games from memdb")
		return
	}

	logrus.WithFields(logrus.Fields{
		"deleted": deleted,
	}).Debug("deleted games from memdb")

	for _, game := range gameListResponse.Results {
		if !n.BotGames && game.BotGame {
			continue
		}
		if game.Width != n.BoardSize {
			continue
		}

		game.MedianRating = (game.White.Ratings.Overall.Rating + game.Black.Ratings.Overall.Rating) / 2

		if game.MedianRating < n.MinMedianRating {
			continue
		}

		if err := txn.Insert("games", game); err != nil {
			logrus.WithFields(logrus.Fields{
				"game_id": game.ID,
				"error":   err,
			}).Warn("could not store in memdb")
			continue
		} else {
			logrus.WithFields(logrus.Fields{
				"game":          game.ID,
				"white_rating":  game.White.Ratings.Overall.Rating,
				"black_rating":  game.Black.Ratings.Overall.Rating,
				"median_rating": game.MedianRating,
			}).Debug("stored in memdb")
		}
	}
	txn.Commit()

	created, updated, deleted := 0, 0, 0
	for _, c := range txn.Changes() {
		if c.Created() {
			created++

			game, ok := c.After.(*websocket.Game)
			if !ok {
				logrus.WithFields(logrus.Fields{
					"table":  c.Table,
					"change": "created",
				}).Warn("expected *websocket.Game")
			}

			n.NotifyGames[game.ID] = game
		}
		if c.Deleted() {
			deleted++
		}
		if c.Updated() {
			updated++
		}
	}
	logrus.WithFields(logrus.Fields{
		"created": created,
		"updated": updated,
		"deleted": deleted,
	}).Debug("memdb changes")
}

func gameStr(game *websocket.Game) string {
	return fmt.Sprintf(
		"%v (%v) vs %v (%v): https://online-go.com/game/%v",
		game.Black.Username,
		game.Black.Ratings.Overall.Rating,
		game.White.Username,
		game.White.Ratings.Overall.Rating,
		game.ID,
	)
}
