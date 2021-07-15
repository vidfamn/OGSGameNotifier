package storage

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/vidfamn/OGSGameNotifier/internal/api"
)

type PlayerOverallRatingIndexer struct{}

func (PlayerOverallRatingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of args %d, expected 1", len(args))
	}
	i, ok := args[0].(float64)
	if !ok {
		return nil, fmt.Errorf("wrong type for arg %T, expected string", args[0])
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(i))

	return b, nil
}

func (PlayerOverallRatingIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	p, ok := raw.(*api.Player)
	if !ok {
		return false, nil, fmt.Errorf("wrong type for arg %T, expected api.Player", raw)
	}
	if p.Ratings.Overall.Rating == 0 {
		return false, nil, nil
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(p.Ratings.Overall.Rating))

	return true, b, nil
}
