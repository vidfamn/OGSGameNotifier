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
	Token  string
	Client *gosocketio.Client

	clockDrift   float64
	clockLatency float64

	pingTicker *time.Ticker
}

func NewOGSWebSocket(token string) (*OGSWebSocket, error) {
	waitForConnection := make(chan struct{}, 1)

	c, err := gosocketio.Dial(
		gosocketio.GetUrl("online-go.com", 443, true),
		transport.GetDefaultWebsocketTransport(),
	)
	if err != nil {
		return nil, errors.New("could not connect websocket: " + err.Error())
	}

	ows := &OGSWebSocket{
		Token:  token,
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

		if ows.pingTicker != nil {
			ows.pingTicker.Stop()
		}
	})

	c.On(gosocketio.OnError, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id":    c.Id(),
			"error": err,
		}).Error("websocket error")

		if ows.pingTicker != nil {
			ows.pingTicker.Stop()
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

		ows.clockLatency = float64(latencyMs) / 1000
		ows.clockDrift = float64(driftMs) / 1000
	})

	go func() {
		for range ows.pingTicker.C {
			msg := &PingRequest{
				Client:  time.Now().UnixNano() / int64(time.Millisecond),
				Drift:   ows.clockDrift,
				Latency: ows.clockLatency,
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

	return ows, nil
}

func (ows *OGSWebSocket) GameListRequest(msg *GameListQueryRequest, timeout time.Duration) (*GameListQueryResponse, error) {
	logrus.WithFields(logrus.Fields{
		"method":  "gamelist/query",
		"message": fmt.Sprintf("%+v", msg),
	}).Debug("sending message")

	resp, err := ows.Client.Ack("gamelist/query", msg, timeout)
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

// func (ows *OGSWebSocket) SendGameListCountSubscribe(ch *gosocketio.Channel) {
// 	logrus.WithFields(logrus.Fields{
// 		"method":  "gamelist/count/subscribe",
// 		"message": "",
// 	}).Debug("sending message")

// 	if err := ch.Emit("gamelist/count/subscribe", ""); err != nil {
// 		logrus.WithFields(logrus.Fields{
// 			"method": "gamelist/count/subscribe",
// 			"error":  err,
// 		}).Error("could not send message")
// 	}
// }
