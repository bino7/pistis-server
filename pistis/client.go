package pistis

import (
	"fmt"
	"time"
)

func startClient(name, mqttServer string) (*server, error) {
	c, e := NewServer(name, mqttServer)
	if e != nil {
		return nil, e
	}
	if e := c.mqttChannel.Subscribe(fmt.Sprint("pistis/", name, "/m")); e != nil {
		panic(e)
	}
	/*not needed*/
	/*c.RegisterHandler("offer",handleTransmission)
	c.RegisterHandler("candidate",handleTransmission)
	c.RegisterHandler("answer",handleTransmission)*/
	c.RegisterHandler("offline", handleOffline)
	c.Start()
	return c, nil
}

/*func handleTransmission(s *server, m Message) {
	ds := server(m.Dst)
	if ds == nil {
		dstNotOpen(s)
		return
	}
	ds.mqttChannel.Input() <- m
}

func dstNotOpen(s *server) {

}*/

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
