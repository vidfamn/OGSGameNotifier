package storage

import "github.com/hashicorp/go-memdb"

func Schema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"game": {
				Name: "game",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:         "id",
						AllowMissing: false,
						Unique:       true,
						Indexer: &memdb.IntFieldIndex{
							Field: "ID",
						},
					},
					"white.ratings.overall.rating": {
						Name:         "white.ratings.overall.rating",
						AllowMissing: false,
						Unique:       false,
						Indexer:      &WhiteOverallRatingIndexer{},
					},
					"black.ratings.overall.rating": {
						Name:         "black.ratings.overall.rating",
						AllowMissing: false,
						Unique:       false,
						Indexer:      &BlackOverallRatingIndexer{},
					},
				},
			},
		},
	}
}
