package tasks

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
)

const Mqtt TaskType = "mqtt"

func init() {
	registry.Add(Mqtt, MqttHandlerFactory)
}

func MqttHandlerFactory(conf map[string]any) (TaskHandler, error) {
	handler := MqttHandler{
		Port:    1883,
		Timeout: timeout,
	}
	err := util.DecodeOther(conf, &handler)

	if err == nil && handler.Port == 0 {
		err = errors.New("missing port")
	}

	return &handler, err
}

type MqttHandler struct {
	Port    int
	Topic   string
	Timeout time.Duration
}

func (h *MqttHandler) Test(log *util.Logger, in ResultDetails) []ResultDetails {
	addr := net.JoinHostPort(in.IP, strconv.Itoa(h.Port))

	recv := make(chan bool, 1)
	ctx := context.Background()

	client, err := mqtt.Connect(ctx, autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{{Scheme: "mqtt", Host: addr}},
		CleanStartOnInitialConnection: true,
		ConnectTimeout:                timeout,
		ClientConfig: paho.ClientConfig{
			PacketTimeout: timeout,
			OnPublishReceived: mqtt.OnPublish(func(*paho.Publish) {
				select {
				case recv <- true:
				default:
				}
			}),
		},
	}, timeout)
	if err != nil {
		return nil
	}
	defer func() { _ = client.Disconnect(ctx) }()

	if h.Topic == "" {
		return nil
	}

	_, _ = client.Subscribe(ctx, &paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{{Topic: h.Topic, QoS: 1}},
	})

	select {
	case <-recv:
		out := in
		out.Topic = h.Topic
		return []ResultDetails{out}
	case <-time.After(timeout):
		return nil
	}
}
