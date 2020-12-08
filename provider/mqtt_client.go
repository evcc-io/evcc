package provider

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/andig/evcc/util"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	connectTimeout = 2 * time.Second
	publishTimeout = 2 * time.Second
)

// MqttClientID created unique mqtt client id
func MqttClientID() string {
	pid := rand.Int31()
	return fmt.Sprintf("evcc-%d", pid)
}

// MqttConfig is the public configuration
type MqttConfig struct {
	Broker   string
	User     string
	Password string
}

// MqttClient capsulates mqtt publish/subscribe functions
type MqttClient struct {
	log      *util.Logger
	mux      sync.Mutex
	Client   mqtt.Client
	broker   string
	Qos      byte
	listener map[string]func(string)
}

// NewMqttClient creates new publisher for paho
func NewMqttClient(
	log *util.Logger,
	broker string,
	user string,
	password string,
	clientID string,
	qos byte,
) (*MqttClient, error) {
	broker = util.DefaultPort(broker, 1883)
	log.INFO.Printf("connecting %s at %s", clientID, broker)

	mc := &MqttClient{
		log:      log,
		broker:   broker,
		Qos:      qos,
		listener: make(map[string]func(string)),
	}

	options := mqtt.NewClientOptions()
	options.AddBroker(broker)
	options.SetUsername(user)
	options.SetPassword(password)
	options.SetClientID(clientID)
	options.SetCleanSession(true)
	options.SetAutoReconnect(true)
	options.SetOnConnectHandler(mc.ConnectionHandler)
	options.SetConnectionLostHandler(mc.ConnectionLostHandler)
	options.SetConnectTimeout(connectTimeout)

	client := mqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("error connecting: %w", token.Error())
	}

	mc.Client = client
	return mc, nil
}

// ConnectionLostHandler logs cause of connection loss as warning
func (m *MqttClient) ConnectionLostHandler(client mqtt.Client, reason error) {
	m.log.ERROR.Printf("%s connection lost: %v", m.broker, reason.Error())
}

// ConnectionHandler restores listeners
func (m *MqttClient) ConnectionHandler(client mqtt.Client) {
	m.log.DEBUG.Printf("%s connected", m.broker)

	m.mux.Lock()
	defer m.mux.Unlock()

	for topic, l := range m.listener {
		m.log.TRACE.Printf("%s subscribe %s", m.broker, topic)
		go m.listen(topic, l)
	}
}

// Publish synchronously pulishes payload using client qos
func (m *MqttClient) Publish(topic string, retained bool, payload interface{}) error {
	token := m.Client.Publish(topic, m.Qos, retained, payload)
	if token.WaitTimeout(publishTimeout) {
		return token.Error()
	}
	return nil
}

// Listen validates uniqueness and registers and attaches listener
func (m *MqttClient) Listen(topic string, callback func(string)) {
	m.mux.Lock()
	if _, ok := m.listener[topic]; ok {
		m.log.FATAL.Fatalf("%s: duplicate listener not allowed", topic)
	}
	m.listener[topic] = callback
	m.mux.Unlock()

	m.listen(topic, callback)
}

// listen attaches listener to topic
func (m *MqttClient) listen(topic string, callback func(string)) {
	token := m.Client.Subscribe(topic, m.Qos, func(c mqtt.Client, msg mqtt.Message) {
		s := string(msg.Payload())
		if len(s) > 0 {
			callback(s)
		}
	})
	m.WaitForToken(token)
}

// WaitForToken synchronously waits until token operation completed
func (m *MqttClient) WaitForToken(token mqtt.Token) {
	if token.WaitTimeout(publishTimeout) {
		if token.Error() != nil {
			m.log.ERROR.Printf("error: %s", token.Error())
		}
	} else {
		m.log.DEBUG.Println("timeout")
	}
}
