package push

import (
	"errors"

	"github.com/gregdel/pushover"
)

// PushOver implements the pushover messenger
type PushOver struct {
	app        *pushover.Pushover
	recipients []string
}

type pushOverConfig struct {
	App        string
	Recipients []string
	Events     map[string]EventTemplate
}

// NewPushOverMessenger creates new pushover messenger
func NewPushOverMessenger(app string, recipients []string) (*PushOver, error) {
	if app == "" {
		return nil, errors.New("pushover: missing app name")
	}

	m := &PushOver{
		app:        pushover.New(app),
		recipients: recipients,
	}

	return m, nil
}

// Send sends to all receivers
func (m *PushOver) Send(title, msg string) {
	message := pushover.NewMessageWithTitle(msg, title)

	for _, id := range m.recipients {
		go func(id string) {
			log.Debug("pushover: sending to %s", id)

			recipient := pushover.NewRecipient(id)
			if _, err := m.app.SendMessage(message, recipient); err != nil {
				log.Error("%v", err)
			}
		}(id)
	}
}
