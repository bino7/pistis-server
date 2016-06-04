package pistis

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

type MqttChannel interface {
	Topics() []string
	Input() chan <- Message
	Messages() <-chan Message
	Errors() <-chan error
	Subscribe(topic string) error
	UnSubscribe(topic string) error
	Close()
}

type mqttChannel struct {
	mqttClient *MQTT.Client
	topics     []string
	input      chan Message
	messages   chan Message
	errors     chan error
	done       chan bool
}

func NewMqttChannel(server string) (*mqttChannel, error) {
	opts := MQTT.NewClientOptions().AddBroker(server)
	client := MQTT.NewClient(opts)
	c := &mqttChannel{
		mqttClient:client,
		topics:make([]string, 0),
		input:make(chan Message),
		messages:make(chan Message),
		errors:make(chan error),
		done:make(chan bool),
	}
	if token := c.mqttClient.Connect(); token.Wait() && token.Error() {
		return nil, token.Error()
	}
	started := make(chan bool)
	go func() {
		defer c.mqttClient.Disconnect(0)
		started <- true
		for {
			select {
			case msg := <-c.input:
				if token := c.mqttClient.Publish(msg.Topic(), msg.Qos(), msg.Retained(), msg.Payload()); token.Wait() && token.Error() != nil {
					c.errors <- token.Error()
				}
			case <-c.done:
				return
			}
		}
	}()
	<-started
	return c
}

func (c *mqttChannel)Topics() []string {
	return c.topics
}

func (c *mqttChannel)Input() chan <- *MQTT.Message {
	return c.input
}
func (c *mqttChannel)Messages() <-chan *MQTT.Message {
	return c.messages
}

func (c *mqttChannel)Errors() <-chan error {
	return c.errors
}
func (c *mqttChannel)Subscribe(topic string) error {
	if token := c.mqttClient.Subscribe(topic, 0, func(client *MQTT.Client, msg MQTT.Message) {
		c.messages <- fromMQTTMessage(msg)
	}); token.Wait()&&token.Error() != nil {
		return token.Error()
	}

	return nil
}
func (c *mqttChannel)UnSubscribe(topic string) error {
	if token := c.mqttClient.Unsubscribe(topic);token.Wait()&&token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c *mqttChannel)Close() {
	c.done <- true
}


