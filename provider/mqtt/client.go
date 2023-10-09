package mqtt

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"strings"
	"sync"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
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
	Insecure bool
}

// Client encapsulates mqtt publish/subscribe functions
type Client struct {
	log      *util.Logger
	mux      sync.Mutex
	Client   paho.Client
	broker   string
	Qos      byte
	listener map[string][]func(string)
}

type Option func(*paho.ClientOptions)

const secure = "tls://"

// NewClient creates new Mqtt publisher
func NewClient(log *util.Logger, broker, user, password, clientID string, qos byte, insecure bool, opts ...Option) (*Client, error) {
	broker, isSecure := strings.CutPrefix(broker, secure)

	// strip schema as it breaks net.SplitHostPort
	broker = util.DefaultPort(broker, 1883)
	if isSecure {
		broker = secure + broker
	}

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
	options.SetConnectTimeout(request.Timeout)
	options.SetOrderMatters(false)

	if insecure {
		options.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	}

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
	log.INFO.Printf("connecting %s at %s", clientID, mc.broker)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("error connecting: %w", token.Error())
	}

	mc.Client = client

	return mc, nil
}

// ConnectionLostHandler logs cause of connection loss as warning
func (m *Client) ConnectionLostHandler(client paho.Client, reason error) {
	m.log.ERROR.Printf("%s connection lost: %v", m.broker, reason.Error())
}

// ConnectionHandler restores listeners
func (m *Client) ConnectionHandler(client paho.Client) {
	m.log.DEBUG.Printf("%s connected", m.broker)

	m.mux.Lock()
	defer m.mux.Unlock()

	for topic := range m.listener {
		m.log.DEBUG.Printf("%s subscribe %s", m.broker, topic)
		go m.listen(topic)
	}
}

// Publish synchronously publishes payload using client qos
func (m *Client) Publish(topic string, retained bool, payload interface{}) error {
	m.log.TRACE.Printf("send %s: '%v'", topic, payload)
	token := m.Client.Publish(topic, m.Qos, retained, payload)
	go m.WaitForToken("send", topic, token)
	return nil
}

// Listen validates uniqueness and registers and attaches listener
func (m *Client) Listen(topic string, callback func(string)) {
	m.mux.Lock()
	m.listener[topic] = append(m.listener[topic], callback)
	m.mux.Unlock()

	m.listen(topic)
}

// ListenSetter creates a /set listener that resets the payload after handling
func (m *Client) ListenSetter(topic string, callback func(string) error) {
	topic += "/set"
	m.Listen(topic, func(payload string) {
		if err := callback(payload); err != nil {
			m.log.ERROR.Printf("set %s: %v", topic, err)
		}
		if err := m.Publish(topic, true, ""); err != nil {
			m.log.ERROR.Printf("clear: %s: %v", topic, err)
		}
	})
}

// listen attaches listener to topic
func (m *Client) listen(topic string) {
	token := m.Client.Subscribe(topic, m.Qos, func(c paho.Client, msg paho.Message) {
		payload := string(msg.Payload())
		m.log.TRACE.Printf("recv %s: '%v'", topic, payload)
		if len(payload) > 0 {
			m.mux.Lock()
			callbacks := m.listener[topic]
			m.mux.Unlock()

			for _, cb := range callbacks {
				cb(payload)
			}
		}
	})
	go m.WaitForToken("subscribe", topic, token)
}

// WaitForToken synchronously waits until token operation completed
func (m *Client) WaitForToken(action, topic string, token paho.Token) {
	err := api.ErrTimeout
	if token.WaitTimeout(request.Timeout) {
		err = token.Error()
	}
	if err != nil {
		m.log.ERROR.Printf("%s: %s: %v", action, topic, err)
	}
}
