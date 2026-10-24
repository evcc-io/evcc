// Ported from shuette42/ecoflow-energy-ha (MIT), see the package documentation.

package ecoflow

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/google/uuid"
)

const brokerURL = "wss://mqtt-e.ecoflow.com:8084/mqtt"

// AppKey is one (appKey, secret) pair of the EcoFlow web app. The broker only
// accepts client IDs signed with such a pair.
type AppKey struct {
	Key, Secret string
}

// clientID builds a web app client ID WEB_{uuid}_{userId}_{appKey}_{ms}_{MD5}.
// The broker rejects a reused ID after a disconnect, so every connect gets a new one.
func clientID(userID string, app AppKey) string {
	base := "WEB_" + uuid.NewString() + "_" + userID
	ts := fmt.Sprint(time.Now().UnixMilli())
	sum := md5.Sum([]byte(app.Secret + base + ts))
	return base + "_" + app.Key + "_" + ts + "_" + strings.ToUpper(fmt.Sprintf("%x", sum))
}

// pahoTransport is the Transport on the EcoFlow app broker.
type pahoTransport struct {
	client paho.Client
}

// dial connects to the app broker with the session's MQTT credentials. onLost
// is called when the established connection drops.
func dial(log *util.Logger, session *Session, app AppKey, onLost func(error)) (Transport, error) {
	opts := paho.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(clientID(session.UserID, app)).
		SetUsername(session.Account).
		SetPassword(session.Password).
		SetCleanSession(true).
		SetKeepAlive(60 * time.Second).
		SetConnectTimeout(request.Timeout).
		SetAutoReconnect(false) // a reused client ID is rejected

	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		onLost(err)
	})

	client := paho.NewClient(opts)
	if err := connect(client, request.Timeout); err != nil {
		return nil, err
	}
	log.DEBUG.Printf("connected to %s", brokerURL)

	return &pahoTransport{client: client}, nil
}

// connect connects client within timeout. On failure it disconnects the
// client, which would otherwise finish connecting in the background and hold
// a second session of the account.
func connect(client paho.Client, timeout time.Duration) error {
	token := client.Connect()
	if err := waitTimeout(token, timeout); err != nil {
		client.Disconnect(0)
		if ct, ok := token.(*paho.ConnectToken); ok {
			switch code := ct.ReturnCode(); code {
			case packets.Accepted, packets.ErrNetworkError, packets.ErrProtocolViolation:
				// no CONNACK from the broker
			default:
				err = &ConnackError{Code: code}
			}
		}
		return fmt.Errorf("connect: %w", err)
	}
	return nil
}

func waitTimeout(token paho.Token, timeout time.Duration) error {
	if !token.WaitTimeout(timeout) {
		return errors.New("timeout")
	}
	return token.Error()
}

func wait(ctx context.Context, token paho.Token) error {
	select {
	case <-token.Done():
		return token.Error()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close disconnects from the broker.
func (t *pahoTransport) Close() {
	t.client.Disconnect(250)
}

// Subscribe implements Transport.
func (t *pahoTransport) Subscribe(topic string, qos byte, handler func([]byte)) error {
	return waitTimeout(t.client.Subscribe(topic, qos, func(_ paho.Client, msg paho.Message) {
		handler(msg.Payload())
	}), request.Timeout)
}

// Publish implements Transport.
func (t *pahoTransport) Publish(ctx context.Context, topic string, qos byte, payload []byte) error {
	return wait(ctx, t.client.Publish(topic, qos, false, payload))
}
