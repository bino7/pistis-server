package pistis

import (
	"sync"
	"fmt"
	"time"
	"encoding/json"
)

type Server interface {
	Name() string
	Start() error
	Stop() error
	IsDone() bool
	RegisterHandler(string, interface{})
}

type Handler func(*server, *Message)

type server struct {
	mqttServer  string
	mqttChannel MqttChannel
	name        string
	input       chan *Message
	done        chan bool
	handlers    map[string]Handler
	mu          sync.Mutex
}

func NewServer(name, mqttServer string) (*server, error) {
	c, err := NewMqttChannel(mqttServer)
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	return &server{
		mqttServer:mqttServer,
		mqttChannel:c,
		name:name,
		input:make(chan *Message),
		done:make(chan bool),
		handlers:make(map[string]Handler),
		mu:mu,
	}, nil
}

func (s *server)Name() string {
	return s.name
}

func (s *server)Input() chan <- *Message {
	return s.input
}

func (s *server)Start() {
	started := make(chan bool)
	s.mqttChannel.Subscribe(topic(s))
	go func() {
		started <- true
		for {
			select {
			case msg := <-s.mqttChannel.Messages():
				s.handle(msg)
			case msg := <-s.input:
				s.handle(msg)
			case <-s.done:
				return
			}
		}
	}()
	<-started
}

func topic(s *server) string {
	if s.name == "pistis" {
		return "pistis/m"
	} else {
		return fmt.Sprint("pistis/", s.name, "/m")
	}
}

func (s *server)Stop() {
	s.done <- true
}

func (s *server)RegisterHandler(t string, h Handler) {
	if t != "" && h != nil {
		s.handlers[t] = h
	}
}

func (s *server)handle(m *Message) {
	if m == nil {
		return
	}
	h := s.handlers[m.Type]
	if h == nil {
		fmt.Println("unhandle message", m)
	}
	h(s, m)
}

func (s *server) IsDone() bool {
	select {
	case <-s.done:
		return true

	default:
	}

	return false
}

var pistis *server

func Start(mqttServer string) *server {
	s, e := NewServer("pistis", mqttServer)
	if e != nil {
		panic(e)
	}
	if e := s.mqttChannel.Subscribe("pistis/m"); e != nil {
		panic(e)
	}
	s.RegisterHandler("open", open)
	s.Start()
	pistis = s
	return s
}

func open(s *server, m *Message) {
	if client(m.Src) != nil {
		return
	}
	c, e := startClient(m.Src, s.mqttServer)
	if e != nil {
		panic(e)
	}

	cs := make([]string, 0)
	for _, cl := range clients {
		cs = append(cs, cl.name)
	}
	clientsJSON, err := json.Marshal(cs)
	if err != nil {
		panic(err)
	}

	addClient(c.name, c)
	fmt.Println("new client", c.name)

	pistis.mqttChannel.Input() <- &Message{
		TimeStamp :time.Now().Unix(),
		Type      :"online",
		Src       :"pistis",
		Dst       :"pistis",
		Payload   :c.name,
	}

	c.mqttChannel.Input() <- &Message{
		TimeStamp :time.Now().Unix(),
		Type      :"clients",
		Src       :"pistis",
		Dst       :c.name,
		Payload   :string(clientsJSON),
	}

}

var clients = make(map[string]*server)
var mu sync.RWMutex

func addClient(name string, s *server) {
	mu.Lock()
	defer mu.Unlock()
	clients[name] = s
}

func client(name string) *server {
	mu.RLock()
	defer mu.RUnlock()
	return clients[name]
}

func (s *server)send(m *Message) {
	s.mqttChannel.Input() <- m
}





