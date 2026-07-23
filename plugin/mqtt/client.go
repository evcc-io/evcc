package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/sync/semaphore"
)

// Instance is the paho Mqtt client singleton
var Instance *Client

const parallelInflightLimit int64 = 32

// ClientID created unique mqtt client id
func ClientID() string {
	return fmt.Sprintf("evcc-%d", rand.Int32())
}

// Config is the public configuration
type Config struct {
	Broker     string `json:"broker"`
	User       string `json:"user"`
	Password   string `json:"password"`
	ClientID   string `json:"clientID"`
	Insecure   bool   `json:"insecure"`
	CaCert     string `json:"caCert"`
	ClientCert string `json:"clientCert"`
	ClientKey  string `json:"clientKey"`
}

// Client encapsulates mqtt publish/subscribe functions
type Client struct {
	log      *util.Logger
	mux      sync.Mutex
	client   paho.Client
	broker   string
	Qos      byte
	listener map[string][]func(string)
	inflight *semaphore.Weighted

	connMu     sync.Mutex
	connCtx    context.Context
	connCancel context.CancelFunc
}

type Option func(*paho.ClientOptions)

const secure = "tls://"

// NewClient creates new Mqtt publisher
func NewClient(log *util.Logger, broker, user, password, clientID string, qos byte, insecure bool, caCert, clientCert, clientKey string, opts ...Option) (*Client, error) {
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
		inflight: semaphore.NewWeighted(parallelInflightLimit),
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
	options.SetWriteTimeout(request.Timeout)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
	}
	if caCert != "" {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(caCert)); !ok {
			return nil, fmt.Errorf("failed to add ca cert to cert pool")
		}
		tlsConfig.RootCAs = caCertPool
	}
	if clientCert != "" && clientKey != "" {
		clientKeyPair, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to add client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientKeyPair}
	}
	options.SetTLSConfig(tlsConfig)

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

	mc.client = client

	return mc, nil
}

// connContext returns a context that is cancelled on disconnect and replaced
// on every reconnect. Pending publishes use it to bail out if the connection
// that scheduled them has gone away.
func (m *Client) connContext() context.Context {
	m.connMu.Lock()
	defer m.connMu.Unlock()
	if m.connCtx == nil {
		return context.Background()
	}
	return m.connCtx
}

func (m *Client) renewConnContext() {
	m.connMu.Lock()
	defer m.connMu.Unlock()
	if m.connCancel != nil {
		m.connCancel()
	}
	m.connCtx, m.connCancel = context.WithCancel(context.Background())
}

func (m *Client) cancelConnContext() {
	m.connMu.Lock()
	defer m.connMu.Unlock()
	if m.connCancel != nil {
		m.connCancel()
	}
}

// ConnectionLostHandler logs cause of connection loss as warning
func (m *Client) ConnectionLostHandler(client paho.Client, reason error) {
	m.log.ERROR.Printf("%s connection lost: %v", m.broker, reason.Error())
	m.cancelConnContext()
}

// ConnectionHandler restores listeners
func (m *Client) ConnectionHandler(client paho.Client) {
	m.log.DEBUG.Printf("%s connected", m.broker)
	m.renewConnContext()

	m.mux.Lock()
	topics := make([]string, 0, len(m.listener))
	for topic := range m.listener {
		topics = append(topics, topic)
	}
	m.mux.Unlock()

	// Resubscribe sequentially to avoid bursting the broker right after reconnect.
	go func() {
		for _, topic := range topics {
			m.log.DEBUG.Printf("%s subscribe %s", m.broker, topic)
			token := m.listen(topic)
			if !token.WaitTimeout(request.Timeout) {
				m.log.ERROR.Printf("subscribe %s: timeout", topic)
			} else if err := token.Error(); err != nil {
				m.log.ERROR.Printf("subscribe %s: %v", topic, err)
			}
		}
	}()
}

// Cleanup recursively removes a topic
func (m *Client) Cleanup(topic string, retained bool) error {
	timer := time.NewTimer(time.Second)

	statusTopic := topic + "/status"
	if !m.client.Subscribe(topic+"/#", m.Qos, func(c paho.Client, msg paho.Message) {
		if len(msg.Payload()) == 0 || msg.Topic() == statusTopic {
			return
		}

		m.log.TRACE.Printf("delete: %s", msg.Topic())
		// Delete synchronously at QoS 0 so the cleanup burst self-throttles
		// to one in-flight message at a time and avoids overwhelming the
		// broker (see #30086).
		token := m.client.Publish(msg.Topic(), 0, true, "")
		token.WaitTimeout(request.Timeout)

		// reset timeout
		timer.Reset(time.Second)
	}).WaitTimeout(request.Timeout) {
		return api.ErrTimeout
	}

	// wait for cleanup to finish
	<-timer.C

	if !m.client.Unsubscribe(topic + "/#").WaitTimeout(request.Timeout) {
		return api.ErrTimeout
	}

	return nil
}

// Publish asynchronously publishes payload using client qos
func (m *Client) Publish(topic string, retained bool, payload string) {
	connCtx := m.connContext()
	go func() {
		ctx, cancel := context.WithTimeout(connCtx, request.Timeout)
		defer cancel()
		if err := m.inflight.Acquire(ctx, 1); err != nil {
			if connCtx.Err() == nil {
				m.log.ERROR.Printf("send %s: %v", topic, err)
			}
			return
		}
		defer m.inflight.Release(1)

		// Bail out if the connection that scheduled this publish has been lost.
		if connCtx.Err() != nil {
			return
		}

		m.log.TRACE.Printf("send %s: '%v'", topic, request.Truncate(payload))
		token := m.client.Publish(topic, m.Qos, retained, payload)

		select {
		case <-connCtx.Done():
			return
		case <-token.Done():
			if err := token.Error(); err != nil {
				m.log.ERROR.Printf("send: %s: %v", topic, err)
			}
		case <-time.After(request.Timeout):
			m.log.ERROR.Printf("send: %s: timeout", topic)
		}
	}()
}

// Listen attaches listener to slice of listeners for given topic
func (m *Client) Listen(topic string, callback func(string)) error {
	m.mux.Lock()
	m.listener[topic] = append(m.listener[topic], callback)
	m.mux.Unlock()

	token := m.listen(topic)

	select {
	case <-time.After(request.Timeout):
		return fmt.Errorf("subscribe: %s: %w", topic, api.ErrTimeout)
	case <-token.Done():
		return nil
	}
}

// ListenSetter creates a /set listener that resets the payload after handling
func (m *Client) ListenSetter(topic string, callback func(string) error) error {
	topic += "/set"
	err := m.Listen(topic, func(payload string) {
		if err := callback(payload); err != nil {
			m.log.ERROR.Printf("set %s: %v", topic, err)
		}
		m.Publish(topic, true, "")
	})
	return err
}

// listen attaches listener to topic
func (m *Client) listen(topic string) paho.Token {
	token := m.client.Subscribe(topic, m.Qos, func(c paho.Client, msg paho.Message) {
		payload := string(msg.Payload())
		m.log.TRACE.Printf("recv %s: '%v'", topic, request.Truncate(payload))
		if len(payload) > 0 {
			m.mux.Lock()
			callbacks := m.listener[topic]
			m.mux.Unlock()

			for _, cb := range callbacks {
				cb(payload)
			}
		}
	})
	return token
}
