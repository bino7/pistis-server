package pistis

import (
	"fmt"
	"time"
	"github.com/google/cayley/quad"
	"github.com/google/cayley"
	g "github.com/google/cayley/graph"
	"errors"
)
var(
	NoClientError = errors.New("client not found")
)

type Client struct {
	UUID string
	User string
}

func (c *Client) quads() []quad.Quad {
	return []quad.Quad{
		cayley.Quad(c.User, "client", c.UUID, ""),
	}
}

func (c *Client) start(mqttServer string) (*server, error) {
	s, e := NewServer(c.UUID,mqttServer)
	if e != nil {
		return nil, e
	}
	if e := s.mqttChannel.Subscribe(fmt.Sprint("pistis/", c.UUID, "/m")); e != nil {
		panic(e)
	}
	s.RegisterHandler("offline", handleOffline)
	s.Start()
	return s, nil
}

func handleOffline(s *server, m *Message) {
	mu.RLock()
	defer func() {
		mu.RUnlock()
		s.Stop()
	}()

	clients[s.name] = nil

	for n, c := range clients {
		c.mqttChannel.Input() <- &Message{
			TimeStamp :time.Now().Unix(),
			Type      :"offline",
			Src       :"pistis",
			Dst       :n,
			Payload   :s.name,
		}
	}
}

func ClientUser(uuid string)(string,error){
	p := cayley.StartPath(store, uuid).In("client")
	it := p.BuildIterator()
	defer it.Close()
	if cayley.RawNext(it) {
		username:=store.NameOf(it.Result())
		return username,nil
	}else{
		return "",NoClientError
	}
}

func (c *Client) Save() error {
	if u,err:=ClientUser(c.UUID);err==NoClientError{
		return store.AddQuadSet(c.quads())
	}else if c.User!=u{
		//update
	}

	return nil
}

func (c *Client) Remove() error{
	tx:=g.NewTransaction()
	for _,quad:=range c.quads(){
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}


