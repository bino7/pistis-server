package pistis

import (
	"fmt"
	"time"
	"github.com/google/cayley/quad"
	"github.com/google/cayley"
	g "github.com/google/cayley/graph"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"sync"
)

var (
	NoClientError = errors.New("client not found")
)

var clients = make(map[string]*Client)
var clients_mu sync.RWMutex

type Client struct {
	*server
	UUID     string
	Username string
	user     *User
}

func (c *Client) quads() []quad.Quad {
	return []quad.Quad{
		cayley.Quad(c.Username, "client", c.UUID, ""),
	}
}

func (c *Client) ensureRunning() error {

	if c.server == nil {
		s, err := NewServer(fmt.Sprint(c.Username, "/", c.UUID), mqtt_server)
		if err != nil {
			return err
		}
		if err := s.mqttChannel.Subscribe(fmt.Sprint("pistis/", s.Name(), "/m")); err != nil {
			return err
		}
		s.RegisterHandler("offline", handleOffline)
		c.server = s
	}

	s := c.server

	if s.IsRunning() == false {
		s.Start()
	}

	return nil
}

func handleOffline(s *server, m *Message) {
	clients_mu.RLock()
	defer func() {
		clients_mu.RUnlock()
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

func checkClient(uuid, username string) bool {
	p := cayley.StartPath(store, username).Out("client")
	it := p.BuildIterator()
	defer it.Close()
	if cayley.RawNext(it) {
		c := store.NameOf(it.Result())
		if uuid == c {
			return true
		}
	}
	return false
}

func getClient(uuid string) (*Client, error) {
	c := clients[uuid]
	if c == nil {
		p := cayley.StartPath(store, uuid).In("client")
		it := p.BuildIterator()
		defer it.Close()
		if cayley.RawNext(it) {
			username := store.NameOf(it.Result())
			c = &Client{UUID:uuid, Username:username}
			clients_mu.Lock()
			defer clients_mu.Unlock()
			clients[uuid] = c
		} else {
			return nil, NoClientError
		}
	}
	err := c.ensureRunning()
	return c, err
}

func (c *Client) save() error {
	if !checkUserName(c.Username) {
		return NoUserError
	}

	tx:=cayley.NewTransaction()
	if checkClient(c.UUID,c.Username)==false{
		p := cayley.StartPath(store, c.UUID).Out(c.Username)
		it := p.BuildIterator()
		defer it.Close()
		if cayley.RawNext(it) {
			un := store.NameOf(it.Result())
			if un != c.Username {
				store.RemoveQuad(cayley.Quad(un, "client", c.UUID, ""))
				u, _ := getUser(un)
				if u != nil {
					u.clients[c.UUID] = nil
				}
			}
		}
		/*store.AddQuadSet(c.quads())*/
		return store.ApplyTransaction(tx)
	}



	if clients[c.UUID] == nil {
		clients_mu.Lock()
		defer clients_mu.Unlock()
		clients[c.UUID] = c
	}
	return nil
}

func (c *Client) remove() error {
	tx := g.NewTransaction()
	for _, quad := range c.quads() {
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}

func (c *Client) token() (*jwt.Token, error) {
	u := c.user
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, MyClaims{
		uuid: "c.UUID",
		username: u.Username,
		password:u.Password,
	})
	return token, nil
}

func (c *Client) sendUserOnlineMsg() {
	c.mqttChannel.Input() <- &Message{
		TimeStamp :time.Now().Unix(),
		Type      :"user-online",
		Src       :"pistis",
		Dst       :c.topic(),
		Payload   :c.user.Username,
	}
}
