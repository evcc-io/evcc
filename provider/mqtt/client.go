package mqtt

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

// Instance is the paho Mqtt client singleton
var Instance *Client

// ClientID created unique mqtt client id
func ClientID() string {
	pid := rand.Int31()
	return fmt.Sprintf("evcc-%d", pid)
}

// Config is the public configuration
type Config struct {
	Broker   string
	User     string
	Password string
}

// Client encapsulates mqtt publish/subscribe functions
type Client struct {
	log      *util.Logger
	mux      sync.Mutex
	Client   mqtt.Client
	broker   string
	Qos      byte
	listener map[string][]func(string)
}

// NewClient creates new Mqtt publisher
func NewClient(log *util.Logger, broker, user, password, clientID string, qos byte) (*Client, error) {
	broker = util.DefaultPort(broker, 1883)
	log.INFO.Printf("connecting %s at %s", clientID, broker)

	mc := &Client{
		log:      log,
		broker:   broker,
		Qos:      qos,
		listener: make(map[string][]func(string)),
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
func (m *Client) ConnectionLostHandler(client mqtt.Client, reason error) {
	m.log.ERROR.Printf("%s connection lost: %v", m.broker, reason.Error())
}

// ConnectionHandler restores listeners
func (m *Client) ConnectionHandler(client mqtt.Client) {
	m.log.DEBUG.Printf("%s connected", m.broker)

	m.mux.Lock()
	defer m.mux.Unlock()

	for topic := range m.listener {
		m.log.TRACE.Printf("%s subscribe %s", m.broker, topic)
		go m.listen(topic)
	}
}

// Publish synchronously pulishes payload using client qos
func (m *Client) Publish(topic string, retained bool, payload interface{}) error {
	token := m.Client.Publish(topic, m.Qos, retained, payload)
	if token.WaitTimeout(publishTimeout) {
		return token.Error()
	}
	return nil
}

// Listen validates uniqueness and registers and attaches listener
func (m *Client) Listen(topic string, callback func(string)) {
	m.mux.Lock()
	m.listener[topic] = append(m.listener[topic], callback)
	m.mux.Unlock()

	m.listen(topic)
}

// listen attaches listener to topic
func (m *Client) listen(topic string) {
	token := m.Client.Subscribe(topic, m.Qos, func(c mqtt.Client, msg mqtt.Message) {
		s := string(msg.Payload())
		if len(s) > 0 {
			m.mux.Lock()
			callbacks := m.listener[topic]
			m.mux.Unlock()

			for _, cb := range callbacks {
				cb(s)
			}
		}
	})
	m.WaitForToken(token)
}

// WaitForToken synchronously waits until token operation completed
func (m *Client) WaitForToken(token mqtt.Token) {
	if token.WaitTimeout(publishTimeout) {
		if token.Error() != nil {
			m.log.ERROR.Printf("error: %s", token.Error())
		}
	} else {
		m.log.DEBUG.Println("timeout")
	}
}
