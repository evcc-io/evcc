package push

import (
	"strings"

	"github.com/andig/evcc/util"
)

// Sender implements message sending
type Sender interface {
	Send(event Event, title, msg string)
}

// EventTemplate is the push message template for an event
type EventTemplate struct {
	Title, Msg string
}

var log = util.NewLogger("push")

// NewMessengerFromConfig creates a new messenger
func NewMessengerFromConfig(typ string, other map[string]interface{}) Sender {
	switch strings.ToLower(typ) {
	case "pushover":
		var cc pushOverConfig
		util.DecodeOther(log, other, &cc)
		return NewPushOverMessenger(cc.App, cc.Recipients)
	case "telegram":
		var cc telegramConfig
		util.DecodeOther(log, other, &cc)
		return NewTelegramMessenger(cc.Token, cc.Chats)
	}

	log.FATAL.Fatalf("unknown messenger type: %s", typ)
	return nil
}
