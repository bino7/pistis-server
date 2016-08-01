package pistis

import (
	"fmt"
	"time"
	g "github.com/google/cayley/graph"
	"github.com/google/cayley/quad"
	"github.com/google/cayley"
	"github.com/dgrijalva/jwt-go"
	"errors"
	"log"
)

var (
	ClientNotExistedError = errors.New("client not existed")
)

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

		c.server = s
	}

	s := c.server

	if s.IsRunning() == false {
		s.Start()
	}

	return nil
}

func (c *Client) token() (*jwt.Token, error) {
	u := c.user
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uuid": c.UUID,
		"username": u.Username,
		"password":u.Password,
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

func checkClient(username,uuid string) bool {
	p := cayley.StartPath(store, username).Out("client")
	it := p.BuildIterator()
	defer it.Close()
	for cayley.RawNext(it) {
		c := store.NameOf(it.Result())
		if uuid == c {
			return true
		}
	}
	return false
}

func getClient(uuid string) (*Client, error) {
	var c *Client
	p := cayley.StartPath(store, uuid).In("client")
	it := p.BuildIterator()
	defer it.Close()
	if cayley.RawNext(it) {
		username := store.NameOf(it.Result())
		c = &Client{UUID:uuid, Username:username}
	} else {
		return nil, ClientNotExistedError
	}
	return c, nil
}

func (c *Client) save() error {
	if !checkUserName(c.Username) {
		return UserNotExistedError
	}

	var oldclient *Client
	tx := cayley.NewTransaction()
	if !checkClient(c.UUID, c.Username) {
		p := cayley.StartPath(store, c.UUID).Out(c.Username)
		it := p.BuildIterator()
		defer it.Close()
		if cayley.RawNext(it) {
			olduser := store.NameOf(it.Result())
			if olduser != c.Username {
				oldclient = &Client{Username:olduser, UUID:c.UUID}
			}
		}
	}

	if oldclient != nil {
		oldclient.remove()
	}

	log.Println("check client:",checkClient(c.Username,c.UUID))

	if !checkClient(c.Username,c.UUID) {
		store.AddQuadSet(c.quads())
	}

	c.user.clients[c.UUID]=c;

	return store.ApplyTransaction(tx)
}

func (c *Client) remove() error {
	tx := g.NewTransaction()
	for _, quad := range c.quads() {
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}
