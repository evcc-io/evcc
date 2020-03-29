package push

import (
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
func NewPushOverMessenger(app string, recipients []string) *PushOver {
	if app == "" {
		log.FATAL.Fatal("pushover: missing app name")
	}

	m := &PushOver{
		app:        pushover.New(app),
		recipients: recipients,
	}

	return m
}

// Send sends to all receivers
func (m *PushOver) Send(event Event, title, msg string) {
	message := pushover.NewMessageWithTitle(msg, title)

	for _, id := range m.recipients {
		go func(id string) {
			log.TRACE.Printf("pushover: sending to %s", id)

			recipient := pushover.NewRecipient(id)
			if _, err := m.app.SendMessage(message, recipient); err != nil {
				log.ERROR.Print(err)
			}
		}(id)
	}
}
