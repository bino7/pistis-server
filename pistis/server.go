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
	RegisterHandler(string, interface{})
}

type server struct {
	mqttServer  string
	mqttChannel MqttChannel
	name        string
	input       chan Message
	done        chan bool
	handlers    map[string]interface{}
	mu          sync.Mutex
}

func NewServer(name, mqttServer string) (*server, error) {
	c, err := NewMqttChannel(mqttServer)
	if err != nil {
		return err
	}
	var mu sync.Mutex
	return &server{
		mqttServer:mqttServer,
		mqttChannel:c,
		name:name,
		done:make(chan bool),
		handlers:make(map[string]interface{}),
		mu:mu,
	}
}

func (s *server)Name() string {
	return s.name
}

func (s *server)Input() chan <- interface{} {
	return s.input
}

func (s *server)Start() {
	started := make(chan bool)
	s.mqttChannel.Subscribe()
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

func (s *server)Stop() {
	s.done <- true
}

func (s *server)RegisterHandler(t string, h interface{}) {
	if s != "" && h != nil {
		s.handlers[t] = h
	}
}

func (s *server)handle(m Message) {
	if m == nil {
		return
	}
	h := s.handlers[m.Type]
	if h == nil {
		fmt.Println("unhandle message", m)
	}
	h(s, m)
}

var pistis *server

func Start(mqttServer string) {
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
}

func open(s *server, m Message) {
	if server(m.Src) != nil {
		return
	}
	c, e := startClient(m.Src, s.mqttServer)
	if e != nil {
		panic(e)
	}
	addClient(m.Src, c)
	c.mqttChannel.Input() <- Message{
		TimeStamp :time.Now().Unix(),
		Type      :"opened",
		Src       :"pistis",
		Dst       :c.name,
		Payload   :"",
	}

	cs := make([]string, 0)
	cs = append(cs, c.Name())
	payload, err := json.Marshal(cs)
	if err != nil {
		panic(err)
	}

	mu.RLock()
	defer mu.RUnlock()

	for n, cl := range clients {
		cl.mqttChannel.Input() <- Message{
			TimeStamp :time.Now().Unix(),
			Type      :"clients",
			Src       :"pistis",
			Dst       :cl.name,
			Payload   :string(payload),
		}
		cs = append(cs, n)
	}
	payload, err = json.Marshal(cs)
	if err != nil {
		panic(err)
	}

	c.mqttChannel.Input() <- Message{
		TimeStamp :time.Now().Unix(),
		Type      :"clients",
		Src       :"pistis",
		Dst       :c.name,
		Payload   :string(payload),
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






