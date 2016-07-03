package pistis

import (
	"sync"
	"fmt"
	"time"
	"encoding/json"
)
var(
	mqtt_server string
	root_server Server
)
type Server interface {
	Name() string
	Start() error
	Stop() error
	IsRunning() bool
	Input() chan <- *Message
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
	s:= &server{
		mqttServer:mqttServer,
		mqttChannel:c,
		name:name,
		input:make(chan *Message),
		done:make(chan bool),
		handlers:make(map[string]Handler),
		mu:mu,
	}

	if name=="pistis" {
		err = s.mqttChannel.Subscribe(fmt.Sprint("pistis/m"))
	}else{
		err = s.mqttChannel.Subscribe(fmt.Sprint("pistis/",s.Name(),"/m"))
	}

	return s,err
}

func (s *server)Name() string {
	return s.name
}

func (s *server)Input() chan <- *Message {
	return s.input
}

func (s *server)Start() {
	started := make(chan bool)
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

func (s *server)topic() string {
	return "pistis/m"
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

func (s *server) IsRunning() bool {
	select {
	case <-s.done:
		return true
	default:
	}
	return false
}

func Start(mqttServer string) *server {
	mqtt_server=mqttServer

	s, err := NewServer("pistis", mqtt_server)
	if err != nil {
		panic(err)
	}
	if err := s.mqttChannel.Subscribe("pistis/m"); err != nil {
		panic(err)
	}
	s.RegisterHandler("open", func(s *server, m *Message){
		type OpenMsg struct {
			Username 	string
			Token 	 	string
		}
		jsonbyte:=([]byte)(m.Payload.(string))
		openMsg:=OpenMsg{"",""}
		if err:=json.Unmarshal(jsonbyte,m);err!=nil{
			s.send(&Message{
				TimeStamp :time.Now().Unix(),
				Type      :"error",
				Src       :"pistis",
				Dst       :m.Src,
				Payload   :err.Error(),
			})
		}

		if !checkToken(openMsg.Token) {
			s.send(&Message{
				TimeStamp :time.Now().Unix(),
				Type      :"forbidden",
				Src       :"pistis",
				Dst       :m.Src,
				Payload   :"token is not Valid",
			})
		}

		u,err:=getUser(openMsg.Username)
		if err!=nil {
			s.send(&Message{
				TimeStamp :time.Now().Unix(),
				Type      :"error",
				Src       :"pistis",
				Dst       :m.Src,
				Payload   :err.Error(),
			})
		}

		u.Input() <- m
	})

	s.Start()
	root_server=s
	return s
}

func (s *server)send(m *Message) {
	s.mqttChannel.Input() <- m
}




