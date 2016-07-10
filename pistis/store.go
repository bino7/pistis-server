package pistis

import (
	"log"
	"github.com/google/cayley"
	"github.com/google/cayley/graph"
	"github.com/google/cayley/graph/bolt"
)

var (
	store_name = "bolt"
	store_path = "ppl.db"
	store *cayley.Handle
)

func initStore() error {
	bolt.Init()
	graph.InitQuadStore("bolt", store_path, nil)
	s, err := cayley.NewGraph("bolt", store_path, nil)
	if err != nil {
		log.Fatal(err)
		return err
	}
	store = s
	return nil
}
