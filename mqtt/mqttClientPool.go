package mqtt

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"container/ring"
	"sync"
)

type MqttClientPool struct {
	url           string
	maxClientNum  int
	clients       *ring.Ring
	subscriptions map[string][]*MqttChannel
	topicClient   map[string]*MQTT.Client
	input         chan *Topublic
	errors        chan error
	mu            sync.Mutex
	wgStopped     sync.WaitGroup
	done          chan interface{}
}

type Topublic struct {
	channel *MqttChannel
	topic   string
	message []byte
}

func NewMqttClientPool(url string, maxClientNum int) *MqttClientPool {
	clients := ring.New(maxClientNum)
	subscriptions := make(map[string][]*MqttChannel)
	topicClient := make(map[string]*MQTT.Client)
	messages := make(chan *Topublic)
	errors := make(chan error)
	var mu sync.Mutex
	var wgStopped sync.WaitGroup
	done := make(chan interface{})
	return &MqttClientPool{
		url:url,
		maxClientNum:maxClientNum,
		clients:clients,
		subscriptions:subscriptions,
		topicClient:topicClient,
		input:messages,
		errors:errors,
		mu:mu,
		wgStopped:wgStopped,
		done:done,
	}
}

func newMQTTClient(url string) *MQTT.Client {
	opts := MQTT.NewClientOptions().AddBroker(url)
	c := MQTT.NewClient(opts)
	return c
}

func (this *MqttClientPool)subscribe(topic string, channel *MqttChannel) error {
	defer func() {
		this.mu.Unlock()
	}()
	this.mu.Lock()

	if this.subscriptions[topic] == nil {
		r := this.clients.Next()
		if r.Value == nil {
			c := newMQTTClient(this.url)
			r.Value = c
			this.wgStopped.Add(1)
			started := make(chan bool)
			go func() {
				defer func() {
					this.wgStopped.Done()
				}()
				started <- true
				for {
					select {
					case toPublic := <-this.input:
						if token := c.Publish(toPublic.topic, 0, false, toPublic.message);
						token.Wait() && token.Error() != nil {
							toPublic.channel.errors <- token.Error()
							this.errors <- token.Error()
						}
					case <-this.done:
						return
					}
				}
			}()
			<-started
		}
		c := r.Value

		chs := make([]*MqttChannel, 0)
		chs = append(chs, channel)
		this.subscriptions[topic] = chs
		if token := c.(*MQTT.Client).Subscribe(topic, 0, func(client *MQTT.Client, msg MQTT.Message) {
			for _, ch := range this.subscriptions[topic] {
				ch.messages <- msg
			}
		}); token.Wait() && token.Error() != nil {
			channel.errors <- token.Error()
			return token.Error()
		}

	}else {
		this.subscriptions[topic] = append(this.subscriptions[topic], channel)
	}

	return nil
}

func (this *MqttClientPool)unsubscribe(topic string, channel *MqttChannel) error {
	defer func() {
		this.mu.Unlock()
	}()
	this.mu.Lock()

	if this.subscriptions[topic] == nil {
		return nil
	}

	removeFromSubscriptions(this.subscriptions[topic], channel)

	if len(this.subscriptions[topic]) == 0 {
		c := this.topicClient[topic]
		if token := c.Unsubscribe("topic"); token.Wait() && token.Error() != nil {
			this.errors <- token.Error()
			return token.Error()
		}
		this.topicClient[topic] = nil
	}

	return nil
}

func removeFromSubscriptions(slice []*MqttChannel, toRemove *MqttChannel) {
	for i := 0; i < len(slice); i++ {
		if slice[i] == toRemove {
			removeFromSliceByIndex(slice, i)
			return
		}
	}
}

func removeFromSliceByIndex(slice []*MqttChannel, i int) {
	if len(slice) < i {
		return
	}

	slice = append(slice[:i], slice[i + 1:]...)
}

func (this *MqttClientPool)Input() chan <- *Topublic {
	return this.input
}

func (this *MqttClientPool)Stop() {
	close(this.done)
	this.wgStopped.Wait()
}





