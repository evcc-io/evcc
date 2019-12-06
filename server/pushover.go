package server

import "github.com/gregdel/pushover"

// PushOver implements the pushover messenger
type PushOver struct {
	app        *pushover.Pushover
	recipients []*pushover.Recipient
}

// NewMessenger creates new pushover messenger
func NewMessenger(app string, recipients []string) *PushOver {
	po := &PushOver{
		app: pushover.New(app),
	}

	for _, r := range recipients {
		po.recipients = append(po.recipients, pushover.NewRecipient(r))
	}

	return po
}

// Send sends to all receivers
func (po *PushOver) Send(sender, msg, title string) {
	message := pushover.NewMessageWithTitle(msg, title)
	message.DeviceName = sender

	for _, recipient := range po.recipients {
		go func(recipient *pushover.Recipient) {
			_, err := po.app.SendMessage(message, recipient)
			if err != nil {
				log.ERROR.Print(err)
			}
		}(recipient)
	}
}
