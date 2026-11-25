package niko

import (
	"cmp"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

var (
	log         = util.NewLogger("niko")
	redactToken = log.RotatingSlot()
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", getDevices)

	service.Register("niko", mux)
}

// smartPlugOptions maps the smart plugs of a device list to options
func smartPlugOptions(devices []Device) []service.Option {
	res := make([]service.Option, 0, len(devices))
	for _, d := range devices {
		if d.Type != TypeSmartPlug {
			continue
		}

		o := service.Option{
			Value: d.Uuid,
			Label: cmp.Or(strings.TrimSpace(d.Name), d.Uuid),
		}
		if status, ok := d.Status(); ok {
			o.Hint = map[bool]string{true: "On", false: "Off"}[status]
		}
		res = append(res, o)
	}

	return res
}

func getDevices(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	token := q.Get("token")

	// the logger is shared, a rotating slot keeps the redaction list from growing while typing
	redactToken(token)

	devices, err := listDevices(q.Get("host"), cmp.Or(q.Get("user"), "hobby"), token)
	if err != nil {
		// form incomplete or controller unreachable, the ui polls while typing
		log.DEBUG.Printf("devices: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-control", "max-age=30")
	_ = json.NewEncoder(w).Encode(smartPlugOptions(devices))
}

// listDevices uses a short-lived connection to not register a broker for every token typed in the ui
func listDevices(host, user, token string) ([]Device, error) {
	if host == "" || token == "" {
		return nil, errors.New("missing host or token")
	}

	res := make(chan []Device, 1)

	u, err := url.Parse(Broker(host))
	if err != nil {
		return nil, err
	}

	// requests are limited by the packet timeout
	ctx := context.Background()

	client, err := mqtt.Connect(ctx, autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		TlsCfg:                        &tls.Config{InsecureSkipVerify: true},
		CleanStartOnInitialConnection: true,
		ConnectTimeout:                request.Timeout,
		ConnectUsername:               user,
		ConnectPassword:               []byte(token),
		ClientConfig: paho.ClientConfig{
			ClientID:      mqtt.ClientID(),
			PacketTimeout: request.Timeout,
			OnPublishReceived: mqtt.OnPublish(func(m *paho.Publish) {
				var msg Message
				if err := json.Unmarshal(m.Payload, &msg); err == nil && msg.Method == MethodList {
					select {
					case res <- msg.Devices():
					default:
					}
				}
			}),
		},
	}, request.Timeout)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Disconnect(ctx) }()

	if _, err := client.Subscribe(ctx, &paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{{Topic: TopicRsp, QoS: 1}},
	}); err != nil {
		return nil, err
	}

	if _, err := client.Publish(ctx, &paho.Publish{Topic: TopicCmd, QoS: 1, Payload: []byte(ListRequest())}); err != nil {
		return nil, err
	}

	select {
	case devices := <-res:
		return devices, nil
	case <-time.After(request.Timeout):
		return nil, api.ErrTimeout
	}
}
