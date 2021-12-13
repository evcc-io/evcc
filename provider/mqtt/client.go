package mqtt

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logx"
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
	ClientID string
}

// Client encapsulates mqtt publish/subscribe functions
type Client struct {
	log      logx.Logger
	mux      sync.Mutex
	Client   paho.Client
	broker   string
	Qos      byte
	listener map[string][]func(string)
}

type Option func(*paho.ClientOptions)

// NewClient creates new Mqtt publisher
func NewClient(log logx.Logger, broker, user, password, clientID string, qos byte, opts ...Option) (*Client, error) {
	broker = util.DefaultPort(broker, 1883)

	mc := &Client{
		log:      log,
		Qos:      qos,
		listener: make(map[string][]func(string)),
	}

	options := paho.NewClientOptions()
	options.AddBroker(broker)
	options.SetUsername(user)
	options.SetPassword(password)
	options.SetClientID(clientID)
	options.SetCleanSession(true)
	options.SetAutoReconnect(true)
	options.SetOnConnectHandler(mc.ConnectionHandler)
	options.SetConnectionLostHandler(mc.ConnectionLostHandler)
	options.SetConnectTimeout(connectTimeout)

	// additional options
	for _, o := range opts {
		o(options)
	}

	client := paho.NewClient(options)

	or := client.OptionsReader()
	mc.broker = fmt.Sprintf("%v", or.Servers())
	if len(or.Servers()) == 1 {
		mc.broker = or.Servers()[0].String()
	}
	logx.Info(log, "msg", fmt.Sprintf("connecting %s at %s", clientID, broker))

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("error connecting: %w", token.Error())
	}

	mc.Client = client

	return mc, nil
}

// ConnectionLostHandler logs cause of connection loss as warning
func (m *Client) ConnectionLostHandler(client paho.Client, reason error) {
	logx.Error(m.log, "msg", fmt.Sprintf("%s connection lost: %v", m.broker, reason.Error()))
}

// ConnectionHandler restores listeners
func (m *Client) ConnectionHandler(client paho.Client) {
	logx.Debug(m.log, "msg", "connected", "broker", m.broker)

	m.mux.Lock()
	defer m.mux.Unlock()

	for topic := range m.listener {
		logx.Debug(m.log, "msg", "subscribe", "broker", m.broker, "topic", topic)
		go m.listen(topic)
	}
}

// Publish synchronously publishes payload using client qos
func (m *Client) Publish(topic string, retained bool, payload interface{}) error {
	logx.Trace(m.log, "topic", topic, "send", payload)
	token := m.Client.Publish(topic, m.Qos, retained, payload)
	if token.WaitTimeout(publishTimeout) {
		return token.Error()
	}
	return api.ErrTimeout
}

// Listen validates uniqueness and registers and attaches listener
func (m *Client) Listen(topic string, callback func(string)) {
	m.mux.Lock()
	m.listener[topic] = append(m.listener[topic], callback)
	m.mux.Unlock()

	m.listen(topic)
}

// ListenSetter creates a /set listener that resets the payload after handling
func (m *Client) ListenSetter(topic string, callback func(string)) {
	m.Listen(topic, func(payload string) {
		callback(payload)
		if err := m.Publish(topic, true, ""); err != nil {
			logx.Error(m.log, "clear", err)
		}
	})
}

// listen attaches listener to topic
func (m *Client) listen(topic string) {
	token := m.Client.Subscribe(topic, m.Qos, func(c paho.Client, msg paho.Message) {
		payload := string(msg.Payload())
		logx.Trace(m.log, "topic", topic, "recv", payload)
		if len(payload) > 0 {
			m.mux.Lock()
			callbacks := m.listener[topic]
			m.mux.Unlock()

			for _, cb := range callbacks {
				cb(payload)
			}
		}
	})
	m.WaitForToken(token)
}

// WaitForToken synchronously waits until token operation completed
func (m *Client) WaitForToken(token paho.Token) {
	if token.WaitTimeout(publishTimeout) {
		if token.Error() != nil {
			logx.Error(m.log, "error", token.Error())
		}
	} else {
		logx.Debug(m.log, "msg", "timeout")
	}
}
