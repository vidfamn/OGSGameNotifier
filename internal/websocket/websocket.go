package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	gosocketio "github.com/ambelovsky/gosf-socketio"
	"github.com/ambelovsky/gosf-socketio/transport"
	"github.com/sirupsen/logrus"
)

type OGSWebSocket struct {
	Client *gosocketio.Client

	ClockDrift   float64
	ClockLatency float64

	pingTicker *time.Ticker
}

func NewOGSWebSocket() (*OGSWebSocket, error) {
	waitForConnection := make(chan struct{}, 1)
	defer close(waitForConnection)

	c, err := gosocketio.Dial(
		gosocketio.GetUrl("online-go.com", 443, true),
		transport.GetDefaultWebsocketTransport(),
	)
	if err != nil {
		return nil, errors.New("could not connect websocket: " + err.Error())
	}

	ogs := &OGSWebSocket{
		Client: c,

		pingTicker: time.NewTicker(25 * time.Second),
	}

	c.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id": c.Id(),
		}).Debug("websocket connected")

		waitForConnection <- struct{}{}
	})

	c.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id": c.Id(),
		}).Debug("websocket disconnected")

		if ogs.pingTicker != nil {
			ogs.pingTicker.Stop()
		}
		select {
		case <-waitForConnection:
		default:
			waitForConnection <- struct{}{}
		}
	})

	c.On(gosocketio.OnError, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id":    c.Id(),
			"error": err,
		}).Error("websocket error")

		if ogs.pingTicker != nil {
			ogs.pingTicker.Stop()
		}
		select {
		case <-waitForConnection:
		default:
			waitForConnection <- struct{}{}
		}
	})

	c.On("net/pong", func(ch *gosocketio.Channel, msg PongResponse) {
		logrus.WithFields(logrus.Fields{
			"method":  "net/pong",
			"message": fmt.Sprintf("%+v", msg),
		}).Debug("received message")

		nowMs := time.Now().UnixNano() / int64(time.Millisecond)
		latencyMs := nowMs - msg.Client
		driftMs := (nowMs - latencyMs/2) - msg.Server

		ogs.ClockLatency = float64(latencyMs) / 1000
		ogs.ClockDrift = float64(driftMs) / 1000
	})

	go func() {
		for range ogs.pingTicker.C {
			msg := &PingRequest{
				Client:  time.Now().UnixNano() / int64(time.Millisecond),
				Drift:   ogs.ClockDrift,
				Latency: ogs.ClockLatency,
			}

			logrus.WithFields(logrus.Fields{
				"method":  "net/ping",
				"message": msg,
				"id":      c.Id(),
			}).Debug("sending message")

			if err := c.Emit("net/ping", msg); err != nil {
				logrus.WithError(err).Error("could not send message")
			}
		}
	}()

	<-waitForConnection

	return ogs, nil
}

// GameListRequest fetches the game list from the websocket.
func (ogs *OGSWebSocket) GameListRequest(msg *GameListQueryRequest, timeout time.Duration) (*GameListQueryResponse, error) {
	logrus.WithFields(logrus.Fields{
		"method":  "gamelist/query",
		"message": fmt.Sprintf("%+v", msg),
	}).Debug("sending message")

	resp, err := ogs.Client.Ack("gamelist/query", msg, timeout)
	if err != nil {
		return nil, errors.New("could not send message: " + err.Error())
	}

	response := &GameListQueryResponse{}
	if err := json.Unmarshal([]byte(resp), response); err != nil {
		return nil, errors.New("could not unmarshal: " + err.Error())
	}

	logrus.WithFields(logrus.Fields{
		"method":   "gamelist/query",
		"response": fmt.Sprintf("%+v", response),
	}).Debug("received message")

	return response, nil
}
