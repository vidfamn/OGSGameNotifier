package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	icon "github.com/vidfamn/OGSGameNotifier/assets"
	"github.com/vidfamn/OGSGameNotifier/internal/storage"
	"github.com/vidfamn/OGSGameNotifier/internal/websocket"

	"github.com/gen2brain/beeep"
	"github.com/gen2brain/dlgs"
	"github.com/getlantern/systray"
	"github.com/hashicorp/go-memdb"
	"github.com/sirupsen/logrus"
)

var (
	Application string = "OGSGameNotifier"

	// Overridden at compile time on make build
	Version string = "dev"
	Build   string = "dev"

	SettingsFile string = ".settings"
	LogFile      string = "errors.log"
)

type Settings struct {
	ProGames        bool    `json:"pro_games"`
	BotGames        bool    `json:"bot_games"`
	MinMedianRating float64 `json:"min_median_rating"`
	BoardSize       int     `json:"board_size"`
}

type Notifier struct {
	OGS      *websocket.OGSWebSocket
	DB       *memdb.MemDB
	Settings Settings

	NotifyGames map[int64]*websocket.Game
}

func main() {
	version := flag.Bool("version", false, "prints application version")
	logLevel := flag.String("log-level", "info", "log level: debug, error, warn, info")
	flag.Parse()

	// Checks if TTY is detected, if not logs will be written to LogFile
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		logrus.SetFormatter(&logrus.TextFormatter{})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		f, err := os.Create(LogFile)
		if err != nil {
			logrus.Panic(err)
		}
		defer f.Close()
		logrus.SetOutput(f)
	}

	switch *logLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	if *version {
		logrus.WithFields(logrus.Fields{
			"application": Application,
			"version":     Version,
			"build":       Build,
		}).Info("application info")
		return
	}

	ogs, err := websocket.NewOGSWebSocket()
	if err != nil {
		logrus.Panic(err)
		return
	}

	db, err := memdb.NewMemDB(storage.Schema())
	if err != nil {
		logrus.Panic(err)
		return
	}

	notifier := &Notifier{
		OGS: ogs,
		DB:  db,
		Settings: Settings{
			ProGames:        true,
			BotGames:        false,
			MinMedianRating: 2300, // ~5d
			BoardSize:       19,
		},
		NotifyGames: map[int64]*websocket.Game{},
	}

	if err := notifier.loadSettings(); err != nil {
		logrus.Panic(err)
	}

	go notifier.pollingLoop()

	systray.Run(onReady(notifier), onExit)
}

func onReady(notifier *Notifier) func() {
	return func() {
		systray.SetIcon(icon.Data)
		systray.SetTitle("OGSGameNotifier")
		systray.SetTooltip("OGSGameNotifier")

		settings := systray.AddMenuItem("Settings", "Settings")
		go func() {
			for {
				<-settings.ClickedCh

				// Approximate rating to rank
				selectedRating, ok, err := dlgs.List("Settings", "Game min median rating:", []string{
					"1900 ~1d", "2000 ~2d", "2100 ~3d", "2200 ~4d", "2300 ~5d", "2400 ~6d", "2500 ~7d", "2600 ~8d", "2700 ~9d",
				})
				if err != nil {
					logrus.Panic(err)
				}
				if selectedRating == "" || !ok {
					continue
				}

				minRating, err := strconv.ParseFloat(selectedRating[:4], 64)
				if err != nil {
					logrus.Panic(err)
				}

				notifier.Settings.MinMedianRating = minRating
				if err := notifier.saveSettings(); err != nil {
					logrus.Panic(err)
				}
			}
		}()

		proGames := systray.AddMenuItemCheckbox("Pro games", "Pro games", true)
		if notifier.Settings.ProGames {
			proGames.Check()
		}

		go func() {
			for {
				<-proGames.ClickedCh
				if proGames.Checked() {
					proGames.Uncheck()
				} else {
					proGames.Check()
				}

				notifier.Settings.ProGames = proGames.Checked()
				if err := notifier.saveSettings(); err != nil {
					logrus.Panic(err)
				}
			}
		}()

		botGames := systray.AddMenuItemCheckbox("Bot games", "Bot games", false)
		if notifier.Settings.BotGames {
			botGames.Check()
		}

		go func() {
			for {
				<-botGames.ClickedCh
				if botGames.Checked() {
					botGames.Uncheck()
				} else {
					botGames.Check()
				}

				notifier.Settings.BotGames = botGames.Checked()
				if err := notifier.saveSettings(); err != nil {
					logrus.Panic(err)
				}
			}
		}()

		systray.AddSeparator()

		quit := systray.AddMenuItem("Quit", "Quit")
		go func() {
			<-quit.ClickedCh
			systray.Quit()
		}()
	}
}

func onExit() {}

func (n *Notifier) saveSettings() error {
	b, _ := json.Marshal(n.Settings)

	if err := ioutil.WriteFile(SettingsFile, b, 0644); err != nil {
		return errors.New("could not write settings file: " + err.Error())
	}

	return nil
}

func (n *Notifier) loadSettings() error {
	// Create a new settings file with current settings if does not exist
	if _, err := os.Stat(SettingsFile); os.IsNotExist(err) {
		f, err := os.Create(SettingsFile)
		if err != nil {
			return errors.New("could not create settings file: " + err.Error())
		}
		defer f.Close()

		b, _ := json.Marshal(n.Settings)
		if _, err = f.Write(b); err != nil {
			return errors.New("could not write settings file: " + err.Error())
		}

		return nil
	}

	// Read and apply stored settings from settings file
	f, err := os.Open(SettingsFile)
	if err != nil {
		return errors.New("could not open settings file: " + err.Error())
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&n.Settings); err != nil {
		return errors.New("could not unmarshal settings: " + err.Error())
	}

	return nil
}

func (n *Notifier) pollingLoop() {
	var stopChan = make(chan os.Signal, 2)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	pollingTicker := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-pollingTicker.C:
			n.updateGameList()

			// Notify new games
			for _, game := range n.NotifyGames {
				logrus.WithFields(logrus.Fields{
					"id":   game.ID,
					"game": gameStr(game),
				}).Debug("sending notification")

				err := beeep.Notify(
					fmt.Sprintf("OGS Game started (%v)", ratingToRank(game.MedianRating)),
					gameStr(game),
					"assets/notification_icon.png",
				)
				if err != nil {
					logrus.WithError(err).Error("could not send notification")
					continue
				}
			}

			// Notifications sent, clear the list
			n.NotifyGames = map[int64]*websocket.Game{}

			// All active games
			txn := n.DB.Txn(false)
			it, err := txn.Get("games", "id")
			if err != nil {
				logrus.WithError(err).Error("could not get games from memdb")
				continue
			}

			for obj := it.Next(); obj != nil; obj = it.Next() {
				_, ok := obj.(*websocket.Game)
				if !ok {
					logrus.Error("expected *websocket.Game")
					continue
				}
			}

		case <-stopChan:
			n.OGS.Close()
			pollingTicker.Stop()
			systray.Quit()
			return
		}
	}
}

func (n *Notifier) updateGameList() {
	gameListResponse, err := n.OGS.GameListRequest(&websocket.GameListQueryRequest{
		List:   "live",
		SortBy: "rank",
		Where: map[string]interface{}{
			"width": n.Settings.BoardSize,
		},
		From:  0,
		Limit: 100,
	}, time.Second*5)
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
		if game.Width != n.Settings.BoardSize {
			continue
		}
		if !n.Settings.BotGames && game.BotGame {
			continue
		}

		if n.Settings.ProGames && (game.Black.Professional || game.White.Professional) {
			game.MedianRating = 2400
		} else {
			game.MedianRating = (game.White.Ratings.Overall.Rating + game.Black.Ratings.Overall.Rating) / 2
		}

		if game.MedianRating < n.Settings.MinMedianRating {
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
	blackRating := strconv.FormatInt(int64(game.Black.Ratings.Overall.Rating), 10)
	if game.Black.Professional {
		blackRating = "pro"
	}

	whiteRating := strconv.FormatInt(int64(game.White.Ratings.Overall.Rating), 10)
	if game.White.Professional {
		whiteRating = "pro"
	}

	return fmt.Sprintf(
		"%v (%v) vs %v (%v): https://online-go.com/game/%v",
		game.Black.Username,
		blackRating,
		game.White.Username,
		whiteRating,
		game.ID,
	)
}

func ratingToRank(rating float64) string {
	r := int((rating-1800)/100 + float64(0.5))
	return fmt.Sprintf("~%vd", r)
}
