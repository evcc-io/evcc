package messenger

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("homeassistant", NewHAMessengerFromConfig)
}

// haSender abstracts the Home Assistant service call for testability
type haSender interface {
	CallService(domain, service string, data map[string]any) error
}

// HAMessenger implements the Home Assistant messenger
type HAMessenger struct {
	log    *util.Logger
	conn   haSender
	notify string
	data   map[string]any
}

// NewHAMessengerFromConfig creates a new Home Assistant messenger
func NewHAMessengerFromConfig(other map[string]any) (api.Messenger, error) {
	var cc struct {
		URI    string
		Notify string
		Data   map[string]any
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("homeassistant")

	conn, err := homeassistant.NewConnection(log, cc.URI, "")
	if err != nil {
		return nil, err
	}

	return &HAMessenger{
		log:    log,
		conn:   conn,
		notify: cc.Notify,
		data:   cc.Data,
	}, nil
}

// Send sends a notification via Home Assistant
func (m *HAMessenger) Send(title, msg string) {
	var err error
	if m.notify != "" {
		domain, service, _ := strings.Cut(m.notify, ".")
		payload := map[string]any{
			"title":   title,
			"message": msg,
		}
		if len(m.data) > 0 {
			payload["data"] = m.data
		}
		err = m.conn.CallService(domain, service, payload)
		// fall back to new-style notify.send_message for integrations
		// that no longer support the legacy service call (e.g. Telegram in HA 2024+)
		if se, ok := errors.AsType[*request.StatusError](err); ok && se.HasStatus(400) {
			err = m.conn.CallService("notify", "send_message", map[string]any{
				"entity_id": m.notify,
				"title":     title,
				"message":   msg,
			})
		}
	} else {
		err = m.conn.CallService("persistent_notification", "create", map[string]any{
			"title":           title,
			"message":         msg,
			"notification_id": "evcc",
		})
	}
	if err != nil {
		m.log.ERROR.Println(err)
	}
}
