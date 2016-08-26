
package pistis

/*import (
	"fmt"
	"time"
	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/quad"
	"github.com/cayleygraph/cayley/graph"
	"github.com/dgrijalva/jwt-go"
	"errors"
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
		quad.Make(c.Username, "client", c.UUID, ""),
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

func checkClient(username, uuid string) bool {
	p := cayley.StartPath(store, quad.StringToValue(username)).Out("client")
	it := p.BuildIterator()
	defer it.Close()
	for it.Next() {
		c := quad.StringOf(store.NameOf(it.Result()))
		if uuid == c {
			return true
		}
	}
	err := p.Iterate(nil).EachValue(nil, func(value quad.Value) {

	})
	if err != nil {
		log.Fatalln(err)
	}
	return false
}

func getClient(uuid string) (*Client, error) {
	var c *Client
	p := cayley.StartPath(store, quad.StringToValue(uuid)).In("client")
	it := p.BuildIterator()
	defer it.Close()
	if it.Next() {
		username := store.NameOf(it.Result())
		c = &Client{UUID:uuid, Username:username.Native().(string)}
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
		p := cayley.StartPath(store, quad.StringToValue(c.UUID)).Out(c.Username)
		it := p.BuildIterator()
		defer it.Close()
		if it.Next() {
			olduser := store.NameOf(it.Result())
			if olduser.String() != c.Username {
				oldclient = &Client{Username:olduser.String(), UUID:c.UUID}
			}
		}
	}

	if oldclient != nil {
		oldclient.remove()
	}

	log.Println("check client:", checkClient(c.Username, c.UUID))

	if !checkClient(c.Username, c.UUID) {
		store.AddQuadSet(c.quads())
	}

	c.user.clients[c.UUID] = c;

	return store.ApplyTransaction(tx)
}

func (c *Client) remove() error {
	tx := graph.NewTransaction()
	for _, quad := range c.quads() {
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}
*/
