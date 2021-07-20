package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	gosocketio "github.com/ambelovsky/gosf-socketio"
	"github.com/ambelovsky/gosf-socketio/transport"
	"github.com/sirupsen/logrus"
)

// OGSWebSocket is used to communicate with the OGS WebSocket API.
type OGSWebSocket struct {
	ClockDrift   float64
	ClockLatency float64

	client    *gosocketio.Client
	clientMu  sync.RWMutex
	closeChan chan struct{}
}

// NewOGSWebSocket creates a new OGSWebSocket and starts listening. The
// connection is kept alive by ping/pong messages which updates ClockDrift
// and ClockLatency. Returns error on connection error.
func NewOGSWebSocket() (*OGSWebSocket, error) {
	waitForConnection := make(chan struct{}, 1)
	defer close(waitForConnection)

	c, err := gosocketio.Dial(
		gosocketio.GetUrl("online-go.com", 443, true),
		transport.GetDefaultWebsocketTransport(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not connect websocket: %w", err)
	}

	ogs := &OGSWebSocket{
		client:    c,
		clientMu:  sync.RWMutex{},
		closeChan: make(chan struct{}, 1),
	}

	ogs.registerHandlers(waitForConnection)
	go ogs.pingTickerLoop()

	<-waitForConnection

	return ogs, nil
}

func (ogs *OGSWebSocket) Close() {
	ogs.client.Close()
	ogs.closeChan <- struct{}{}
}

func (ogs *OGSWebSocket) pingTickerLoop() {
	// Send initial ping message then start the loop
	msg := &PingRequest{Client: time.Now().UnixNano() / int64(time.Millisecond)}
	if err := ogs.PingRequest(msg); err != nil {
		logrus.WithError(err).Error("could not send message")
	}

	pingTicker := time.NewTicker(time.Second * 25)
	defer pingTicker.Stop()

	for {
		select {
		case <-pingTicker.C:
			msg := &PingRequest{
				Client:  time.Now().UnixNano() / int64(time.Millisecond),
				Drift:   ogs.ClockDrift,
				Latency: ogs.ClockLatency,
			}
			if err := ogs.PingRequest(msg); err != nil {
				logrus.WithError(err).Error("could not send message")
			}

		case <-ogs.closeChan:
			return
		}
	}
}

func (ogs *OGSWebSocket) registerHandlers(waitForConnection chan struct{}) {
	ogs.client.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id": c.Id(),
		}).Debug("websocket connected")

		waitForConnection <- struct{}{}
	})

	ogs.client.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id": c.Id(),
		}).Debug("websocket disconnected")

		if err := ogs.Reconnect(); err != nil {
			logrus.WithError(err).Error("could not connect to websocket")
		}
	})

	ogs.client.On(gosocketio.OnError, func(c *gosocketio.Channel) {
		logrus.WithFields(logrus.Fields{
			"id": c.Id(),
		}).Debug("websocket error")

		if err := ogs.Reconnect(); err != nil {
			logrus.WithError(err).Error("could not connect to websocket")
		}
	})

	ogs.client.On("net/pong", func(ch *gosocketio.Channel, msg PongResponse) {
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
}

// Reconnect attempts to reconnect the websocket using exponential backoff.
// Blocks API calls through RWLock() until connection is made or aborted.
// Returns nil on success and error on connection error or retry aborted
// (OGSWebSocket.Closed() called).
func (ogs *OGSWebSocket) Reconnect() error {
	ogs.clientMu.Lock()
	defer ogs.clientMu.Unlock()

	factor := 1.5
	max := time.Second * 30
	backoffTicker := make(chan time.Duration, 1)
	backoffTicker <- time.Second * 1
	backoffTickerOpen := true

	for backoffTickerOpen {
		select {
		case d, open := <-backoffTicker:
			if !open {
				backoffTickerOpen = false
				break // channel is closed
			}

			c, err := gosocketio.Dial(
				gosocketio.GetUrl("online-go.com", 443, true),
				transport.GetDefaultWebsocketTransport(),
			)
			if err != nil {
				logrus.WithError(err).Error("could not connect to websocket, retrying...")
			} else {
				ogs.client = c
				close(backoffTicker)
				break // connection established, exit loop
			}

			time.Sleep(d)
			if d >= max {
				backoffTicker <- max
			} else {
				backoffTicker <- time.Duration(float64(d) * factor)
			}
		case <-ogs.closeChan:
			return errors.New("websocket connection retry aborted")
		}

	}

	waitForConnection := make(chan struct{}, 1)

	ogs.registerHandlers(waitForConnection)

	<-waitForConnection

	return nil
}

// PingRequests sends a net/ping request, does not wait for response.
// A net/pong message is expected to be received after this request.
func (ogs *OGSWebSocket) PingRequest(msg *PingRequest) error {
	ogs.clientMu.RLock()
	defer ogs.clientMu.RUnlock()

	logrus.WithFields(logrus.Fields{
		"method":              "net/ping",
		"message":             fmt.Sprintf("%+v", msg),
		"websocket_client_id": ogs.client.Id(),
	}).Debug("sending message")

	if err := ogs.client.Emit("net/ping", msg); err != nil {
		return fmt.Errorf("could not send message: %w", err)
	}

	return nil
}

// GameListRequest fetches the game list from the websocket.
func (ogs *OGSWebSocket) GameListRequest(msg *GameListQueryRequest, timeout time.Duration) (*GameListQueryResponse, error) {
	ogs.clientMu.RLock()
	defer ogs.clientMu.RUnlock()

	logrus.WithFields(logrus.Fields{
		"method":              "gamelist/query",
		"message":             fmt.Sprintf("%+v", msg),
		"websocket_client_id": ogs.client.Id(),
	}).Debug("sending message")

	resp, err := ogs.client.Ack("gamelist/query", msg, timeout)
	if err != nil {
		return nil, fmt.Errorf("could not send message: %w", err)
	}

	response := &GameListQueryResponse{}
	if err := json.Unmarshal([]byte(resp), response); err != nil {
		return nil, fmt.Errorf("could not unmarshal: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"method":          "gamelist/query",
		"response_size":   len(resp),
		"game_list_size":  response.Size,
		"game_list_from":  response.From,
		"game_list_limit": response.Limit,
		"game_list_count": len(response.Results),
	}).Debug("received message")

	return response, nil
}
