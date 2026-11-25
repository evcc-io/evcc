package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/sync/semaphore"
)

// Instance is the paho Mqtt client singleton
var Instance *Client

const parallelInflightLimit int64 = 32

// cleanupMaxDuration stops Cleanup from blocking forever on brokers that never
// stop delivering non-empty messages for the cleaned topic
const cleanupMaxDuration = 30 * time.Second

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
	client   *autopaho.ConnectionManager
	router   *paho.StandardRouter
	broker   string
	Qos      byte
	listener map[string][]func(string)
	inflight *semaphore.Weighted

	connMu     sync.Mutex
	connCtx    context.Context
	connCancel context.CancelFunc
}

type Option func(*autopaho.ClientConfig)

const secure = "tls://"

// NewClient creates new Mqtt publisher
func NewClient(log *util.Logger, broker, user, password, clientID string, qos byte, insecure bool, caCert, clientCert, clientKey string, opts ...Option) (*Client, error) {
	broker, isSecure := strings.CutPrefix(broker, secure)

	// strip schema as it breaks net.SplitHostPort
	broker = util.DefaultPort(broker, 1883)
	if isSecure {
		broker = secure + broker
	} else if !strings.Contains(broker, "://") {
		broker = "mqtt://" + broker
	}

	u, err := url.Parse(broker)
	if err != nil {
		return nil, fmt.Errorf("invalid broker: %w", err)
	}

	mc := &Client{
		log:      log,
		router:   paho.NewStandardRouter(),
		broker:   u.String(),
		Qos:      qos,
		listener: make(map[string][]func(string)),
		inflight: semaphore.NewWeighted(parallelInflightLimit),
	}

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

	// session expiry is left at 0, so every connection starts without broker-side subscriptions
	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		TlsCfg:                        tlsConfig,
		KeepAlive:                     30,
		CleanStartOnInitialConnection: true,
		ConnectTimeout:                request.Timeout,
		ConnectUsername:               user,
		ConnectPassword:               []byte(password),
		OnConnectionUp:                mc.ConnectionHandler,
		OnConnectionDown: func() bool {
			mc.cancelConnContext()
			return true
		},
		ClientConfig: paho.ClientConfig{
			ClientID:      clientID,
			PacketTimeout: request.Timeout,
			OnPublishReceived: OnPublish(func(msg *paho.Publish) {
				mc.router.Route(msg.Packet())
			}),
			OnClientError: func(err error) {
				log.ERROR.Printf("%s connection lost: %v", mc.broker, err)
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				log.ERROR.Printf("%s connection lost: disconnected by broker (reason %d)", mc.broker, d.ReasonCode)
			},
		},
	}

	// additional options
	for _, o := range opts {
		o(&cfg)
	}

	log.INFO.Printf("connecting %s at %s", clientID, mc.broker)

	client, err := Connect(context.Background(), cfg, request.Timeout)
	if err != nil {
		return nil, fmt.Errorf("error connecting: %w", err)
	}

	mc.client = client

	return mc, nil
}

// Connect returns the connection once the broker is reached, failing on the first connect error
func Connect(ctx context.Context, cfg autopaho.ClientConfig, timeout time.Duration) (*autopaho.ConnectionManager, error) {
	errC := make(chan error, 1)
	cfg.OnConnectError = func(err error) {
		select {
		case errC <- err:
		default:
		}
	}

	cm, err := autopaho.NewConnection(ctx, cfg)
	if err != nil {
		return nil, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	upC := make(chan error, 1)
	go func() { upC <- cm.AwaitConnection(waitCtx) }()

	select {
	case err = <-upC:
	case err = <-errC:
	}

	if err != nil {
		// stop the connection manager, it retries forever otherwise
		go func() { _ = cm.Disconnect(context.Background()) }()
		return nil, err
	}

	return cm, nil
}

// OnPublish adapts handler to the client config's publish callbacks
func OnPublish(handler func(*paho.Publish)) []func(paho.PublishReceived) (bool, error) {
	return []func(paho.PublishReceived) (bool, error){
		func(pr paho.PublishReceived) (bool, error) {
			handler(pr.Packet)
			return true, nil
		},
	}
}

// timeout returns a context limiting a single broker request
func timeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), request.Timeout)
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

// ConnectionHandler restores listeners
func (m *Client) ConnectionHandler(_ *autopaho.ConnectionManager, _ *paho.Connack) {
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
			if err := m.subscribe(topic); err != nil {
				m.log.ERROR.Printf("subscribe %s: %v", topic, err)
			}
		}
	}()
}

// Cleanup recursively removes a topic
func (m *Client) Cleanup(topic string, retained bool) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	// callbacks hand topics to the loop below, so no delete runs after cleanup returns
	topics := make(chan string)
	done := make(chan struct{})

	statusTopic := topic + "/status"
	filter := topic + "/#"

	m.router.RegisterHandler(filter, func(msg *paho.Publish) {
		if len(msg.Payload) == 0 || msg.Topic == statusTopic {
			return
		}

		select {
		case topics <- msg.Topic:
		case <-done:
		}
	})
	defer m.router.UnregisterHandler(filter)

	if err := m.subscribe(filter); err != nil {
		close(done)
		return err
	}

	deadline := time.NewTimer(cleanupMaxDuration)
	defer deadline.Stop()

	// wait for cleanup to finish
wait:
	for {
		select {
		case t := <-topics:
			m.log.TRACE.Printf("delete: %s", t)
			// Delete synchronously at QoS 0 so the cleanup burst self-throttles
			// to one in-flight message at a time and avoids overwhelming the
			// broker (see #30086).
			ctx, cancel := timeout()
			_, _ = m.client.Publish(ctx, &paho.Publish{Topic: t, Retain: true})
			cancel()

			// reset timeout
			timer.Reset(time.Second)

		case <-timer.C:
			break wait

		case <-deadline.C:
			m.log.WARN.Printf("cleanup %s: broker still sending retained messages after %v, continuing", topic, cleanupMaxDuration)
			break wait
		}
	}

	// release blocked callbacks before unsubscribing, they hold up paho's message routing
	close(done)

	ctx, cancel := timeout()
	defer cancel()

	_, err := m.client.Unsubscribe(ctx, &paho.Unsubscribe{Topics: []string{filter}})
	return err
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
		// limited by the packet timeout
		_, err := m.client.Publish(connCtx, &paho.Publish{Topic: topic, QoS: m.Qos, Retain: retained, Payload: []byte(payload)})
		if err != nil && connCtx.Err() == nil {
			m.log.ERROR.Printf("send: %s: %v", topic, err)
		}
	}()
}

// Listen attaches listener to slice of listeners for given topic
func (m *Client) Listen(topic string, callback func(string)) error {
	m.mux.Lock()
	first := len(m.listener[topic]) == 0
	m.listener[topic] = append(m.listener[topic], callback)
	m.mux.Unlock()

	// registered outside the lock, the router holds its own lock while running handlers
	if first {
		m.router.RegisterHandler(topic, m.handler(topic))
	}

	if err := m.subscribe(topic); err != nil {
		return fmt.Errorf("subscribe: %s: %w", topic, err)
	}

	return nil
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

// subscribe subscribes topic at the broker, failing if the broker refuses it
func (m *Client) subscribe(topic string) error {
	ctx, cancel := timeout()
	defer cancel()

	_, err := m.client.Subscribe(ctx, &paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{{Topic: topic, QoS: m.Qos}},
	})
	return err
}

// handler dispatches messages to the listeners of topic
func (m *Client) handler(topic string) paho.MessageHandler {
	return func(msg *paho.Publish) {
		payload := string(msg.Payload)
		m.log.TRACE.Printf("recv %s: '%v'", topic, request.Truncate(payload))
		if len(payload) > 0 {
			m.mux.Lock()
			callbacks := m.listener[topic]
			m.mux.Unlock()

			for _, cb := range callbacks {
				cb(payload)
			}
		}
	}
}
