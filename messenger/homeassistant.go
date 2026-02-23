package messenger

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
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
}

// NewHAMessengerFromConfig creates a new Home Assistant messenger
func NewHAMessengerFromConfig(other map[string]any) (api.Messenger, error) {
	var cc struct {
		URI    string
		Notify string
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
	}, nil
}

// Send sends a notification via Home Assistant
func (m *HAMessenger) Send(title, msg string) {
	go func() {
		var err error
		if m.notify != "" {
			domain, service, _ := strings.Cut(m.notify, ".")
			err = m.conn.CallService(domain, service, map[string]any{
				"title":   title,
				"message": msg,
			})
		} else {
			err = m.conn.CallService("persistent_notification", "create", map[string]any{
				"title":           title,
				"message":         msg,
				"notification_id": "evcc",
			})
		}
		if err != nil {
			m.log.ERROR.Printf("homeassistant: %v", err)
		}
	}()
}
