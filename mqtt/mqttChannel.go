package mqtt

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"sync"
)

var PanicHandler func(interface{})

type MqttChannel struct {
	clientPool *MqttClientPool
	topics     []string
	input      chan *Topublic
	messages   chan MQTT.Message
	errors     chan error
	done       chan bool
	mu         sync.Mutex
}

func NewMqttChannel(clientPool *MqttClientPool) (*MqttChannel, error) {
	topics := make([]string, 0)
	input := make(chan *Topublic)
	messages := make(chan MQTT.Message)
	errors := make(chan error)
	done := make(chan bool)
	var mu sync.Mutex
	return &MqttChannel{
		clientPool:clientPool,
		topics:topics,
		input:input,
		messages:messages,
		errors:errors,
		done:done,
		mu:mu,
	}, nil
}

func (this *MqttChannel)Topics() []string {
	return this.topics
}
func (this *MqttChannel)Input() chan<- *Topublic {
	return this.input
}
func (this *MqttChannel)Messages() <-chan MQTT.Message {
	return this.messages
}
func (this *MqttChannel)Errors() <-chan error {
	return this.errors
}
func (this *MqttChannel)Subscribe(topic string) error {
	if err := this.clientPool.subscribe(topic, this); err != nil {
		return err
	}
	this.mu.Lock()
	this.topics = append(this.topics, topic)
	this.mu.Unlock()
	return nil
}
func (this *MqttChannel)Start() error {
	started := make(chan bool)
	go withRecover(func() {
		started <- true
		for {
			select {
			case msg := <-this.input:
				this.clientPool.Input() <- &Topublic{this, msg.topic,msg.message}
			case <-this.done:
				return
			}
		}
	})
	<-started
	return nil
}

func (this *MqttChannel)Close() {
	close(this.done)
}

func withRecover(fn func()) {
	defer func() {
		handler := PanicHandler
		if handler != nil {
			if err := recover(); err != nil {
				handler(err)
			}
		}
	}()

	fn()
}
