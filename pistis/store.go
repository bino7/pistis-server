package pistis

import (
	"github.com/google/cayley"
	"log"
)

var (
	store_name = "bolt"
	store_path = "pistis.db"
	store *cayley.Handle
)

func initStore() error {
	s, err := cayley.NewGraph(store_name, store_path, nil)
	if err != nil {
		log.Fatal(err)
		return err
	}
	store = s
	return nil
}
