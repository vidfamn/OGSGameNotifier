package websocket

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	gosocketio "github.com/ambelovsky/gosf-socketio"
	"github.com/ambelovsky/gosf-socketio/transport"
	"github.com/sirupsen/logrus"
)

func OGSWebSocket(token string) error {
	var clockDrift, clockLatency float64 = 0, 0

	c, err := gosocketio.Dial(
		gosocketio.GetUrl("online-go.com", 443, true),
		transport.GetDefaultWebsocketTransport(),
	)
	if err != nil {
		return errors.New("could not connect websocket: " + err.Error())
	}

	c.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id": c.Id(),
		}).Debug("websocket connected")
	})

	c.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id": c.Id(),
		}).Debug("websocket disconnected")

		os.Interrupt.Signal()
	})

	c.On(gosocketio.OnError, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id":    c.Id(),
			"error": err,
		}).Error("websocket error")
	})

	c.On("gamelist-count", func(ch *gosocketio.Channel, msg gameListCountResponse) {
		logrus.WithFields(logrus.Fields{
			"method":  "gamelist-count",
			"message": fmt.Sprintf("%+v", msg),
		}).Debug("received message")
	})

	c.On("net/pong", func(ch *gosocketio.Channel, msg pongResponse) {
		logrus.WithFields(logrus.Fields{
			"method":  "net/pong",
			"message": fmt.Sprintf("%+v", msg),
		}).Debug("received message")

		nowMs := time.Now().UnixNano() / int64(time.Millisecond)
		latencyMs := nowMs - msg.Client
		driftMs := (nowMs - latencyMs/2) - msg.Server

		clockLatency = float64(latencyMs) / 1000
		clockDrift = float64(driftMs) / 1000
	})

	pingTicker := time.NewTicker(25 * time.Second)
	defer pingTicker.Stop()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-pingTicker.C:
			msg := &pingRequest{
				Client:  time.Now().UnixNano() / int64(time.Millisecond),
				Drift:   clockDrift,
				Latency: clockLatency,
			}

			logrus.WithFields(logrus.Fields{
				"method":  "net/ping",
				"message": msg,
				"id":      c.Id(),
			}).Debug("sending message")

			if err := c.Emit("net/ping", msg); err != nil {
				logrus.WithError(err).Error("could not emit event")
			}

		case <-done:
			c.Close()
			return nil
		}
	}
}

type GameListQueryRequest struct {
	List    string                 `json:"list"`
	SortBy  string                 `json:"sort_by"`
	Where   map[string]interface{} `json:"where"`
	From    int                    `json:"from"`
	Limit   int                    `json:"limit"`
	Channel string                 `json:"channel"`
}

func SendGameListQuery(ch *gosocketio.Channel, msg *GameListQueryRequest) error {
	logrus.WithFields(logrus.Fields{
		"method":  "gamelist/query",
		"message": fmt.Sprintf("%+v", msg),
	}).Debug("sending message")

	return ch.Emit("gamelist/query", msg)
}

func SendGameListCountSubscribe(ch *gosocketio.Channel) {
	logrus.WithFields(logrus.Fields{
		"method":  "gamelist/count/subscribe",
		"message": "",
	}).Debug("sending message")

	if err := ch.Emit("gamelist/count/subscribe", ""); err != nil {
		logrus.WithFields(logrus.Fields{
			"method": "gamelist/count/subscribe",
			"error":  err,
		}).Error("could not send message")
	}
}
