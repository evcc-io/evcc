package push

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/gregdel/pushover"
)

func init() {
	registry.Add("pushover", NewPushOverFromConfig)
}

// PushOver implements the pushover messenger
type PushOver struct {
	log        *util.Logger
	app        *pushover.Pushover
	device     string
	recipients []string
}

// NewPushOverFromConfig creates new pushover messenger
func NewPushOverFromConfig(other map[string]interface{}) (Messenger, error) {
	var cc struct {
		App        string
		Recipients []string
		Devices    []string
		Events     map[string]EventTemplate
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.App == "" {
		return nil, errors.New("missing app name")
	}

	m := &PushOver{
		log:        util.NewLogger("pushover"),
		app:        pushover.New(cc.App),
		device:     strings.Join(cc.Devices, ","),
		recipients: cc.Recipients,
	}

	return m, nil
}

// Send sends to all receivers
func (m *PushOver) Send(title, msg string) {
	message := pushover.NewMessageWithTitle(msg, title)
	message.DeviceName = m.device

	for _, id := range m.recipients {
		go func(id string) {
			m.log.DEBUG.Printf("sending to %s", id)

			recipient := pushover.NewRecipient(id)
			if _, err := m.app.SendMessage(message, recipient); err != nil {
				m.log.ERROR.Print(err)
			}
		}(id)
	}
}
