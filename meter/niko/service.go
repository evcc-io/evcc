package niko

import (
	"cmp"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
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

	options := paho.NewClientOptions()
	options.AddBroker(Broker(host))
	options.SetUsername(user)
	options.SetPassword(token)
	options.SetClientID(mqtt.ClientID())
	options.SetCleanSession(true)
	options.SetConnectTimeout(request.Timeout)
	options.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	client := paho.NewClient(options)
	if err := wait(client.Connect()); err != nil {
		return nil, err
	}
	defer client.Disconnect(0)

	if err := wait(client.Subscribe(TopicRsp, 1, func(_ paho.Client, m paho.Message) {
		var msg Message
		if err := json.Unmarshal(m.Payload(), &msg); err == nil && msg.Method == MethodList {
			select {
			case res <- msg.Devices():
			default:
			}
		}
	})); err != nil {
		return nil, err
	}

	if err := wait(client.Publish(TopicCmd, 1, false, ListRequest())); err != nil {
		return nil, err
	}

	select {
	case devices := <-res:
		return devices, nil
	case <-time.After(request.Timeout):
		return nil, api.ErrTimeout
	}
}

func wait(token paho.Token) error {
	if !token.WaitTimeout(request.Timeout) {
		return api.ErrTimeout
	}
	return token.Error()
}
