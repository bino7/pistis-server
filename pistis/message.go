package pistis

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"fmt"
	"encoding/json"
)

type Message struct {
	TimeStamp int64
	Type      string
	Src       string
	Dst       string
	Payload   interface{}
}

func (m *Message)Duplicate() bool {
	return false
}
func (m *Message)Qos() byte {
	return byte(0)
}
func (m *Message)Retained() bool {
	return false
}
func (m *Message)Topic() string {
	return m.Dst
}
func (m *Message)MessageID() uint16 {
	return uint16(m.TimeStamp)
}
func (m *Message)Marshal() []byte {
	if data, e := json.Marshal(m); e != nil {
		panic(e)
	} else {
		return data
	}
}
func UnMarshal(m MQTT.Message) *Message {
	msg := &Message{}
	if e := json.Unmarshal(m.Payload(), msg); e != nil {
		panic(e)
	}
	return msg
}

func (m *Message) asString() string{
	return fmt.Sprintf("TimeStamp:%s,Type:%s,Src:%s,Dst:%s,Payload:%s",
		m.TimeStamp,m.Type,m.Src,m.Dst,m.Payload)
}