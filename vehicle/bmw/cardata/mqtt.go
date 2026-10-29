package cardata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/evcc-io/evcc/api"
	pluginmqtt "github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

type MqttConnector struct {
	mu            sync.RWMutex
	log           *util.Logger
	subscriptions map[string][]subscription
}

// subscription connects the handler to a single receiver through an unbounded queue
type subscription struct {
	in  chan StreamingMessage
	out <-chan StreamingMessage
}

// queued forwards in to the returned channel through an unbounded fifo queue, so that
// a busy receiver cannot block the mqtt handler. Closing in drains the queue, then closes out.
func queued(in <-chan StreamingMessage) <-chan StreamingMessage {
	out := make(chan StreamingMessage)

	go func() {
		defer close(out)

		var queue []StreamingMessage

		for in != nil || len(queue) > 0 {
			var send chan<- StreamingMessage
			var next StreamingMessage

			if len(queue) > 0 {
				send, next = out, queue[0]
			}

			select {
			case msg, ok := <-in:
				if !ok {
					in = nil
					continue
				}
				queue = append(queue, msg)

			case send <- next:
				queue = queue[1:]
			}
		}
	}()

	return out
}

var (
	mqttMu          sync.Mutex
	mqttConnections = make(map[string]*MqttConnector)
)

func NewMqttConnector(ctx context.Context, log *util.Logger, clientID string, ts oauth2.TokenSource) *MqttConnector {
	return connector(log, clientID, func(v *MqttConnector) { v.run(ctx, ts) })
}

// NewBrokerConnector creates a connector that reads the unmodified CarData stream from a local broker
func NewBrokerConnector(ctx context.Context, log *util.Logger, cc pluginmqtt.Config, topic string) *MqttConnector {
	// prefix keeps broker connectors apart from BMW connectors keyed by client id
	key := fmt.Sprintf("broker:%s.%s:%s:%s", cc.Broker, cc.User, cc.Password, topic)
	return connector(log, key, func(v *MqttConnector) { v.runBroker(ctx, cc, topic) })
}

// connector returns the shared connector for key, starting run for a new one
func connector(log *util.Logger, key string, run func(*MqttConnector)) *MqttConnector {
	mqttMu.Lock()
	defer mqttMu.Unlock()

	if conn, ok := mqttConnections[key]; ok {
		return conn
	}

	v := &MqttConnector{
		log:           log,
		subscriptions: make(map[string][]subscription),
	}

	if !testing.Testing() {
		go run(v)
	}

	mqttConnections[key] = v

	return v
}

// runBroker retries until the broker is first reached, the mqtt client reconnects and resubscribes on its own afterwards
func (v *MqttConnector) runBroker(ctx context.Context, cc pluginmqtt.Config, topic string) {
	bo := backoff.NewExponentialBackOff(backoff.WithInitialInterval(time.Second), backoff.WithMaxInterval(time.Minute), backoff.WithMaxElapsedTime(0))

	client, err := backoff.RetryNotifyWithData(func() (*pluginmqtt.Client, error) {
		return pluginmqtt.RegisteredClientOrDefault(v.log, cc)
	}, backoff.WithContext(bo, ctx), func(err error, _ time.Duration) {
		v.log.ERROR.Println(err)
	})
	if err != nil {
		return
	}

	// listener stays registered and is resubscribed on reconnect, so retrying would duplicate it
	if err := client.Listen(topic, v.brokerHandler); err != nil {
		v.log.ERROR.Println(err)
	}
}

func (v *MqttConnector) Subscribe(vin string) <-chan StreamingMessage {
	v.mu.Lock()
	defer v.mu.Unlock()

	vin = strings.ToUpper(vin)

	sub := subscription{in: make(chan StreamingMessage)}
	sub.out = queued(sub.in)

	v.subscriptions[vin] = append(v.subscriptions[vin], sub)
	v.log.DEBUG.Printf("mqtt subscribe: %s (%d active subscribers)", vin, len(v.subscriptions[vin]))

	return sub.out
}

func (v *MqttConnector) Unsubscribe(vin string, ch <-chan StreamingMessage) {
	v.mu.Lock()
	defer v.mu.Unlock()

	vin = strings.ToUpper(vin)

	subs := v.subscriptions[vin]

	i := slices.IndexFunc(subs, func(sub subscription) bool { return sub.out == ch })
	if i < 0 {
		return
	}

	// queued messages are delivered before out is closed
	close(subs[i].in)

	if subs = slices.Delete(subs, i, i+1); len(subs) == 0 {
		delete(v.subscriptions, vin)
	} else {
		v.subscriptions[vin] = subs
	}
}

func (v *MqttConnector) run(ctx context.Context, ts oauth2.TokenSource) {
	bo := backoff.NewExponentialBackOff(backoff.WithInitialInterval(time.Second), backoff.WithMaxInterval(time.Minute), backoff.WithMaxElapsedTime(0))

	for ctx.Err() == nil {
		time.Sleep(bo.NextBackOff())

		token, err := ts.Token()
		if err != nil {
			if _, ok := errors.AsType[*api.ErrLoginRequired](err); !ok {
				v.log.ERROR.Println(err)
			}

			continue
		}

		if err := v.runMqtt(ctx, token); err != nil {
			v.log.ERROR.Println(err)

			// don't reset backoff
			if errors.Is(err, packets.ErrorRefusedBadUsernameOrPassword) || errors.Is(err, packets.ErrorRefusedNotAuthorised) {
				continue
			}
		}

		bo.Reset()
	}
}

func (v *MqttConnector) runMqtt(ctx context.Context, token *oauth2.Token) error {
	gcid := TokenExtra(token, "gcid")
	idToken := TokenExtra(token, "id_token")

	v.log.DEBUG.Printf("connect streaming (using gcid %s, id_token %s, valid: %v)", gcid, idToken, token.Expiry.Round(time.Second))

	paho := mqtt.NewClient(
		mqtt.NewClientOptions().
			AddBroker(StreamingURL).
			SetAutoReconnect(true).
			SetUsername(gcid).
			SetPassword(idToken))

	timeout := 30 * time.Second
	if t := paho.Connect(); !t.WaitTimeout(timeout) {
		return errors.New("connect timeout")
	} else if err := t.Error(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer paho.Disconnect(1000)

	topic := fmt.Sprintf("%s/+", gcid)

	if t := paho.Subscribe(topic, 0, v.handler); !t.WaitTimeout(timeout) {
		return errors.New("subcribe timeout")
	} else if err := t.Error(); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	v.log.DEBUG.Println("connected streaming")

	ctx, cancel := context.WithDeadline(ctx, token.Expiry)
	defer cancel()

	<-ctx.Done()

	return nil
}

func (v *MqttConnector) handler(_ mqtt.Client, m mqtt.Message) {
	var res StreamingMessage
	if err := json.Unmarshal(m.Payload(), &res); err != nil {
		v.log.ERROR.Println(m.Topic(), string(m.Payload()), err)
		return
	}

	v.log.TRACE.Printf("recv %s: %s", m.Topic(), string(m.Payload()))

	v.dispatch(res)
}

// brokerHandler skips foreign payloads quietly since a local broker topic may carry more than CarData envelopes
func (v *MqttConnector) brokerHandler(payload string) {
	var res StreamingMessage
	if err := json.Unmarshal([]byte(payload), &res); err != nil || res.Vin == "" || len(res.Data) == 0 {
		v.log.DEBUG.Printf("skip invalid message: %s", request.Truncate(payload))
		return
	}

	v.dispatch(res)
}

func (v *MqttConnector) dispatch(res StreamingMessage) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, sub := range v.subscriptions[strings.ToUpper(res.Vin)] {
		sub.in <- res
	}
}
