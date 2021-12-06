package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	icon "github.com/vidfamn/OGSGameNotifier/assets"
	"github.com/vidfamn/OGSGameNotifier/internal/storage"
	"github.com/vidfamn/OGSGameNotifier/internal/websocket"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/hashicorp/go-memdb"
	"github.com/sirupsen/logrus"
)

var (
	Application string = "OGSGameNotifier"

	// Overridden at compile time on make build
	Version string = "dev"
	Build   string = "dev"

	SettingsFile     string = configDirPath(".settings")
	LogFile          string = configDirPath("errors.log")
	NotificationIcon string = binDirPath("assets/notification_icon.png")
)

type Settings struct {
	ProGames        bool    `json:"pro_games"`
	BotGames        bool    `json:"bot_games"`
	MinMedianRating float64 `json:"min_median_rating"`
	CustomMatchRE   string  `json:"custom_match_re"`
	BoardSize       int     `json:"board_size"`
}

type Notifier struct {
	OGS      *websocket.OGSWebSocket
	DB       *memdb.MemDB
	Settings Settings

	NotifyGames map[int64]*websocket.Game

	OpenWindow chan fyne.Window
}

func main() {
	version := flag.Bool("version", false, "prints application version")
	logLevel := flag.String("log-level", "info", "log level: debug, error, warn, info")
	flag.Parse()

	logrus.SetFormatter(&logrus.TextFormatter{})

	// Log to file on windows
	if runtime.GOOS == "windows" {
		f, err := os.OpenFile(LogFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			logrus.WithError(err).Error("could not create logfile")
		} else {
			defer f.Close()

			logrus.SetFormatter(&logrus.JSONFormatter{})
			logrus.SetOutput(f)
		}
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

	openWindow := make(chan fyne.Window)

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
		OpenWindow:  openWindow,
	}

	if err := notifier.loadSettings(); err != nil {
		logrus.Panic(err)
	}

	go notifier.pollingLoop()
	go systray.Run(onReady(notifier), onExit)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	fyneApp := app.New()
	fyneApp.Settings().Theme().Font(fyne.TextStyle{
		Monospace: true,
	})

	go func() {
		for {
			select {
			case w := <-openWindow:
				w.Show()
				w.SetCloseIntercept(func() {
					w.Hide()
				})
			case <-sigs:
				fyneApp.Quit()
				return
			}
		}
	}()

	fyneApp.Run()
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

				minMedianRatingBindStr := binding.BindString(new(string))
				bf := binding.NewFloat()
				bf.Set(float64(notifier.Settings.MinMedianRating))
				bf.AddListener(binding.NewDataListener(func() {
					v, _ := bf.Get()
					notifier.Settings.MinMedianRating = v
					minMedianRatingBindStr.Set(strconv.Itoa(int(notifier.Settings.MinMedianRating)) +
						" " + ratingToRank(float64(notifier.Settings.MinMedianRating)))
				}))

				w := fyne.CurrentApp().NewWindow("Settings")
				w.Resize(fyne.Size{Width: 300, Height: 300})
				buttons := container.NewAdaptiveGrid(2,
					widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
						w.Hide()
					}),
					widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
						if err := notifier.saveSettings(); err != nil {
							logrus.WithFields(logrus.Fields{
								"error": err,
							}).Error("could not save settings")
							dialog.ShowError(err, w)
						} else {
							w.Hide()
						}
					}),
				)

				w.SetContent(container.NewBorder(nil, buttons, nil, nil,
					container.NewAppTabs(
						container.NewTabItem("OGS", container.NewVBox(
							widget.NewLabel("Min median dan rating"),
							widget.NewSliderWithData(1900, 2700, bf),
							container.NewCenter(widget.NewLabelWithData(minMedianRatingBindStr)),
							widget.NewSeparator(),
							// widget.NewLabel("Custom match RegExp"),
							// widget.NewEntryWithData(binding.BindString(&customMatchRE)),
							widget.NewCheckWithData("Pro games", binding.BindBool(&notifier.Settings.ProGames)),
							widget.NewCheckWithData("Bot games", binding.BindBool(&notifier.Settings.BotGames)),
						)),
						container.NewTabItem("IGS", container.NewVBox(
							widget.NewLabel("TODO"),
						)),
					),
				))
				notifier.OpenWindow <- w
			}
		}()

		systray.AddSeparator()

		quit := systray.AddMenuItem("Quit", "Quit")
		go func() {
			<-quit.ClickedCh
			systray.Quit()
			fyne.CurrentApp().Quit()
		}()
	}
}

func onExit() {}

func (n *Notifier) saveSettings() error {
	b, _ := json.Marshal(n.Settings)

	if err := ioutil.WriteFile(SettingsFile, b, 0644); err != nil {
		return fmt.Errorf("could not write settings file: %w", err)
	}

	return nil
}

func (n *Notifier) loadSettings() error {
	// Create a new settings file with current settings if does not exist
	if _, err := os.Stat(SettingsFile); os.IsNotExist(err) {
		f, err := os.Create(SettingsFile)
		if err != nil {
			return fmt.Errorf("could not create settings file: %w", err)
		}
		defer f.Close()

		b, _ := json.Marshal(n.Settings)
		if _, err = f.Write(b); err != nil {
			return fmt.Errorf("could not write settings file: %w", err)
		}

		return nil
	}

	// Read and apply stored settings from settings file
	f, err := os.Open(SettingsFile)
	if err != nil {
		return fmt.Errorf("could not open settings file: %w", err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&n.Settings); err != nil {
		return fmt.Errorf("could not unmarshal settings: %w", err)
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
					NotificationIcon,
				)
				if err != nil {
					logrus.WithError(err).Error("could not send notification")
					continue
				}
			}

			// Notifications sent, clear the list
			n.NotifyGames = map[int64]*websocket.Game{}

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
		List: "live",
		// NOTE: SortBy does not seem to work as intended from server, higher ranked games
		// can occur lower down in the returned list.
		SortBy: "rank",
		// FIXME: This where-field does not seem to work, probably wrong format.
		Where: map[string]interface{}{
			"width":        n.Settings.BoardSize,
			"ranked":       true,
			"in_beginning": true,
		},
		From:  0,
		Limit: 1000,
	}, time.Second*5)
	if err != nil {
		logrus.Error(err)
		return
	}

	if gameListResponse.Size > gameListResponse.Limit {
		logrus.WithFields(logrus.Fields{
			"game_list_list":  gameListResponse.List,
			"game_list_by":    gameListResponse.By,
			"game_list_size":  gameListResponse.Size,
			"game_list_from":  gameListResponse.From,
			"game_list_limit": gameListResponse.Limit,
		}).Warn("incomplete game list response")
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

		if !n.Settings.ProGames && (game.Black.Professional || game.White.Professional) {
			continue
		} else {
			game.MedianRating = (game.White.Ratings.Overall.Rating + game.Black.Ratings.Overall.Rating) / 2

			if game.MedianRating < n.Settings.MinMedianRating {
				continue
			}
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
				"pro_game":      game.Black.Professional || game.White.Professional,
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
				continue
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

// ratingToRank is a simplified and approximate conversion of rating to
// rank.
func ratingToRank(rating float64) string {
	r := int((rating - 1800) / 100)
	if r <= 0 {
		return fmt.Sprintf("~%vk", (r-1)*-1)
	}

	return fmt.Sprintf("~%vd", r)
}
