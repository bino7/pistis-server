package mqtt

import (
	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"errors"
	"time"
)

var mqttClientPool *MqttClientPool
var mqttUrl="localhost:3881"
var maxClientNum=100
var(
	IsStartedError=errors.New("is started")
)

type MqttService struct {
	mqttChannel *MqttChannel
	done chan interface{}
	onMessage MessageHandler
	onTimeout TimeoutHandler
	timeout time.Duration
	session map[string]interface{}
	started bool
	lastAccess time.Time
	ticker *time.Ticker
}

type MessageHandler func(MQTT.Message)
type TimeoutHandler func()chan interface{}

func NewMqttService(timeout string) (*MqttService,error) {
	d,err:=time.ParseDuration(timeout)
	if err!=nil {
		return nil,err
	}
	ticker:=time.NewTicker(d)

	if mqttClientPool==nil {
		mqttClientPool=NewMqttClientPool(mqttUrl,maxClientNum)
	}
	mqttChannel,_:=NewMqttChannel(mqttClientPool)
	done:=make(chan interface{})
	session:=make(map[string]interface{})

	return &MqttService{
		mqttChannel:mqttChannel,
		done:done,
		session:session,
		started:false,
		timeout:d,
		lastAccess:time.Now(),
		ticker:ticker,
	},nil
}

func (this *MqttService)Start(onMessage MessageHandler,onTimeout TimeoutHandler)error{
	if this.started {
		return IsStartedError
	}
	this.onMessage=onMessage
	this.onTimeout=onTimeout
	started:=make(chan bool)
	go func(){
		started <- true
		for{
			select{
			case msg:=<-this.mqttChannel.Messages():
				this.onMessage(msg)
				this.lastAccess =time.Now()
			case <-this.ticker.C:
				if this.lastAccess.Add(this.timeout).Before(time.Now()) {
					<-this.onTimeout()
					this.Stop()
				}
			case <-this.done:
				return
			}
		}
	}()
	<-started
	return nil
}


func (this *MqttService)Stop()error{
	this.ticker.Stop()
	close(this.done)
	return nil
}

func (this *MqttService)Set(key string,value interface{}){
	this.session[key]=value
}

func (this *MqttService)Get(key string)interface{}{
	return this.session[key]
}

func (this *MqttService)GetSession()map[string]interface{}{
	return this.session
}

func (this *MqttService)IsStarted()bool{
	return this.started
}

func (this *MqttService)Send(topic string,msg []byte){
	this.mqttChannel.input <- &Topublic{
		this.mqttChannel,
		topic,
		msg,
	}
}
