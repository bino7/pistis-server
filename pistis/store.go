package pistis

import (
	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/bolt"
)

var (
	store_path = "ppl.db"
	store *cayley.Handle
)

func initStore() error {
	graph.InitQuadStore("bolt", store_path, nil)
	s, err := cayley.NewGraph("bolt", store_path, nil)
	if err != nil {
		log.Fatal(err)
		return err
	}
	store = s
	return nil
}
