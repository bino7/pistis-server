package pistis

/*
import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/golang/protobuf/proto"
	"fmt"
)

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
func (m *Message)GenPayload() []byte {
	data,e:=proto.Marshal(m)
	if e!=nil {
		panic(e)
	}
	return data
}

func fromMQTTMessage(m MQTT.Message) *Message{
	msg:=&Message{}
	if e:=proto.Unmarshal(m.Payload(),msg);e!=nil {
		panic(e)
	}
	return msg
}*/
