package cardata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

type MqttConnector struct {
	mu            sync.RWMutex
	log           *util.Logger
	subscriptions map[string]chan StreamingMessage
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
		subscriptions: make(map[string]chan StreamingMessage),
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

	ch := make(chan StreamingMessage, 1)
	v.subscriptions[vin] = ch

	return ch
}

func (v *MqttConnector) Unsubscribe(vin string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if ch, ok := v.subscriptions[vin]; ok {
		delete(v.subscriptions, vin)
		close(ch)
	}
}

func (v *MqttConnector) run(ctx context.Context, ts oauth2.TokenSource) {
	bo := backoff.NewExponentialBackOff(backoff.WithInitialInterval(time.Second), backoff.WithMaxInterval(time.Minute), backoff.WithMaxElapsedTime(0))

	for ctx.Err() == nil {
		time.Sleep(bo.NextBackOff())

		token, err := ts.Token()
		if err != nil {
			if !tokenError(err) {
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

	v.log.TRACE.Println("recv: " + string(m.Payload()))

	v.mu.RLock()
	defer v.mu.RUnlock()

	if ch, ok := v.subscriptions[res.Vin]; ok {
		ch <- res
	}
}
