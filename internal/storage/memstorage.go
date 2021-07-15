package storage

import "github.com/hashicorp/go-memdb"

func Schema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"player": {
				Name: "player",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:         "id",
						AllowMissing: false,
						Unique:       true,
						Indexer: &memdb.IntFieldIndex{
							Field: "ID",
						},
					},
					"ratings.overall.rating": {
						Name:         "ratings.overall.rating",
						AllowMissing: false,
						Unique:       false,
						Indexer:      &PlayerOverallRatingIndexer{},
					},
				},
			},
		},
	}
}
