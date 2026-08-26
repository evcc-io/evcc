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
	"github.com/evcc-io/evcc/util"
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
	mqttMu.Lock()
	defer mqttMu.Unlock()

	if conn, ok := mqttConnections[clientID]; ok {
		return conn
	}

	v := &MqttConnector{
		log:           log,
		subscriptions: make(map[string][]subscription),
	}

	if !testing.Testing() {
		go v.run(ctx, ts)
	}

	mqttConnections[clientID] = v

	return v
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

	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, sub := range v.subscriptions[strings.ToUpper(res.Vin)] {
		sub.in <- res
	}
}
