package cardata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
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

	go v.run(ctx, ts)

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

func (v *MqttConnector) run(ctx context.Context, ts oauth2.TokenSource) {
	bo := backoff.NewExponentialBackOff(backoff.WithMaxInterval(time.Minute))

	for ctx.Err() == nil {
		time.Sleep(bo.NextBackOff())

		token, err := ts.Token()
		if err != nil {
			if !tokenError(err) {
				v.log.ERROR.Println(err)
			}

			continue
		}

		bo.Reset()

		if err := v.runMqtt(ctx, token); err != nil {
			v.log.ERROR.Println(err)
		}
	}
}

func (v *MqttConnector) runMqtt(ctx context.Context, token *oauth2.Token) error {
	gcid := TokenExtra(token, "gcid")
	idToken := TokenExtra(token, "id_token")

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
	defer paho.Disconnect(0)

	topic := fmt.Sprintf("%s/#", gcid)

	if t := paho.Subscribe(topic, 0, v.handler); !t.WaitTimeout(timeout) {
		return errors.New("subcribe timeout")
	} else if err := t.Error(); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	ctx, cancel := context.WithDeadline(ctx, token.Expiry)
	defer cancel()

	<-ctx.Done()

	return nil
}

func (v *MqttConnector) handler(c mqtt.Client, m mqtt.Message) {
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
