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
var(
	NoClientError = errors.New("client not found")
)

var clients = make(map[string]*Client)
var clients_mu sync.RWMutex

type Client struct {
	*server
	UUID 			string
	Username 	string
	user 			*User
}

func (c *Client) quads() []quad.Quad {
	return []quad.Quad{
		cayley.Quad(c.Username, "client", c.UUID, ""),
	}
}

func (c *Client) ensureRunning() error {
	s := c.server
	if s==nil {
		s1, err := NewServer(fmt.Sprint(c.Username,"/",c.UUID),mqtt_server)
		if err != nil {
			return err
		}
		if err := s.mqttChannel.Subscribe(fmt.Sprint("pistis/", s1.Name(), "/m")); err != nil {
			return err
		}
		s.RegisterHandler("offline", handleOffline)
		c.server=s
	}

	if s.IsRunning()==false {
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

func checkClient(uuid,username string) bool{
	p := cayley.StartPath(store, username).Out("clients")
	it := p.BuildIterator()
	defer it.Close()
	if cayley.RawNext(it) {
		c:=store.NameOf(it.Result())
		return uuid==c
	}
	return false
}

func getClient(uuid string)(*Client,error){
	c:=clients[uuid]
	if c==nil {
		p := cayley.StartPath(store, uuid).In("client")
		it := p.BuildIterator()
		defer it.Close()
		if cayley.RawNext(it) {
			username:=store.NameOf(it.Result())
			c=&Client{uuid,username}
			clients_mu.Lock()
			defer clients_mu.Unlock()
			clients[uuid] = c
		}else{
			return nil,NoClientError
		}
	}
	err := c.ensureRunning()
	return c,err
}

func (c *Client) save() error {
	if u,err:=c.user();err==NoClientError{
		return store.AddQuadSet(c.quads())
	}else if c.Username!=u.Username{
		//update
	}
	if clients[c.UUID]==nil{
		clients_mu.Lock()
		defer clients_mu.Unlock()
		clients[c.UUID] = c
	}
	return nil
}

func (c *Client) remove() error{
	tx:=g.NewTransaction()
	for _,quad:=range c.quads(){
		tx.RemoveQuad(quad)
	}
	return store.ApplyTransaction(tx)
}

func (c *Client) token() (*jwt.Token,error){
	u,err:=c.user()
	if err!=nil {
		return nil,err
	}
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = map[string]interface{}{
		"uuid":c.UUID,
		"username":u.Username,
		"password":u.Password,
	}
	return token,nil
}

func (c *Client) sendUserOnlineMsg(){
		clients[c.UUID]
		c.mqttChannel.Input() <- &Message{
			TimeStamp :time.Now().Unix(),
			Type      :"user-online",
			Src       :"pistis",
			Dst       :c.topic(),
			Payload   :c.user.Username,
		}
}
