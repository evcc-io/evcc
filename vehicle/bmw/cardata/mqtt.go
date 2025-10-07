package cardata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/evcc-io/evcc/util"
	"github.com/golang-jwt/jwt/v5"
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

		if err := v.runMqtt(ctx, token); err != nil {
			v.log.ERROR.Println(err)

			// don't reset backoff
			if errors.Is(err, packets.ErrorRefusedBadUsernameOrPassword) {
				continue
			}
		}

		bo.Reset()
	}
}

func (v *MqttConnector) runMqtt(ctx context.Context, token *oauth2.Token) error {
	gcid := TokenExtra(token, "gcid")
	idToken := TokenExtra(token, "id_token")

	var claims jwt.RegisteredClaims
	parsed, err := jwt.ParseWithClaims(idToken, &claims, nil)
	if err != nil && !errors.Is(err, jwt.ErrTokenUnverifiable) {
		return fmt.Errorf("get %w for %s", err, idToken)
	}
	idExpiry, _ := parsed.Claims.GetExpirationTime()

	v.log.DEBUG.Printf("connect streaming (using gcid %s/ id_token %s, IDT valid: %v, AT valid: %v)", gcid, idToken, idExpiry.Round(time.Second), token.Expiry.Round(time.Second))

	u, _ := url.Parse(StreamingURL)
	topic := fmt.Sprintf("%s/#", gcid)
	recvC := make(chan *paho.Publish, 1)

	conf := autopaho.ClientConfig{
		ServerUrls:      []*url.URL{u},
		ConnectUsername: gcid,
		ConnectPassword: []byte(idToken),
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			v.log.DEBUG.Println("mqtt connected")

			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: topic},
				},
			}); err != nil {
				v.log.ERROR.Printf("mqtt failed to subscribe: %v", err)
			}

			v.log.DEBUG.Println("mqtt subscribed")
		},

		ClientConfig: paho.ClientConfig{
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					recvC <- pr.Packet
					return true, nil
				}},
		},
	}

	conn, err := autopaho.NewConnection(ctx, conf)
	if err != nil {
		return err
	}

	if err := conn.AwaitConnection(ctx); err != nil {
		return err
	}
	defer conn.Disconnect(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-recvC:
				v.handler(msg)
			}
		}
	}()

	ctx, cancel := context.WithDeadline(ctx, token.Expiry)
	defer cancel()

	<-ctx.Done()

	return nil
}

func (v *MqttConnector) handler(m *paho.Publish) {
	var res StreamingMessage
	if err := json.Unmarshal(m.Payload, &res); err != nil {
		v.log.ERROR.Println(m.Topic, string(m.Payload), err)
		return
	}

	v.log.TRACE.Println("recv: " + string(m.Payload))

	v.mu.RLock()
	defer v.mu.RUnlock()

	if ch, ok := v.subscriptions[res.Vin]; ok {
		ch <- res
	}
}
