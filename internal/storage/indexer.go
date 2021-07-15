package storage

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/vidfamn/OGSGameNotifier/internal/websocket"
)

type WhiteOverallRatingIndexer struct{}

func (WhiteOverallRatingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of args %d, expected 1", len(args))
	}
	i, ok := args[0].(float64)
	if !ok {
		return nil, fmt.Errorf("wrong type for arg %T, expected float64", args[0])
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(i))

	return b, nil
}

func (WhiteOverallRatingIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	p, ok := raw.(*websocket.Game)
	if !ok {
		return false, nil, fmt.Errorf("wrong type for arg %T, expected *websocket.Game", raw)
	}
	if p.White.Ratings.Overall.Rating == 0 {
		return false, nil, nil
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(p.White.Ratings.Overall.Rating))

	return true, b, nil
}

type BlackOverallRatingIndexer struct{}

func (BlackOverallRatingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of args %d, expected 1", len(args))
	}
	i, ok := args[0].(float64)
	if !ok {
		return nil, fmt.Errorf("wrong type for arg %T, expected float64", args[0])
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(i))

	return b, nil
}

func (BlackOverallRatingIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	p, ok := raw.(*websocket.Game)
	if !ok {
		return false, nil, fmt.Errorf("wrong type for arg %T, expected *websocket.Game", raw)
	}
	if p.Black.Ratings.Overall.Rating == 0 {
		return false, nil, nil
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(p.Black.Ratings.Overall.Rating))

	return true, b, nil
}
