package pistis

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"fmt"
	"encoding/json"
)

type Message struct{
	TimeStamp int64
	Type 			string
	Src 			string
	Dst 			string
	Payload 	string
}

func (m *Message)Duplicate() bool{
	return false
}
func (m *Message)Qos() byte{
	return byte(0)
}
func (m *Message)Retained() bool{
	return false
}
func (m *Message)Topic() string {
	if m.Dst=="pistis" {
		return m.Dst
	}else{
		return fmt.Sprint("pistis/",m.Dst)
	}
}

func (m *Message)MessageID() uint16 {
	return uint16(m.TimeStamp)
}
func (m *Message)Marshal() []byte {
	if data,e:=json.Marshal(m);e!=nil{
		panic(e)
	}else{
		return data
	}
}

func fromMQTTMessage(m MQTT.Message) *Message{
	msg:=&Message{}
	fmt.Println(m.Payload())
	if e:=json.Unmarshal(m.Payload(),msg);e!=nil{
		panic(e)
	}
	return msg
}