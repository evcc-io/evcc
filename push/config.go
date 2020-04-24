package push

import (
	"strings"

	"github.com/andig/evcc/util"
)

// Cacher duplicates the server.Cacher interface
type Cacher interface {
	All(string) (map[string]interface{}, error)
}

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
func NewMessengerFromConfig(typ string, other map[string]interface{}, cache Cacher) Sender {
	switch strings.ToLower(typ) {
	case "pushover":
		var cc pushOverConfig
		util.DecodeOther(log, other, &cc)
		return NewPushOverMessenger(cc.App, cc.Recipients)
	case "telegram":
		var cc telegramConfig
		util.DecodeOther(log, other, &cc)
		tg := NewTelegramMessenger(cc.Token, cc.Chats)
		tg.Cache = cache
		return tg
	}

	log.FATAL.Fatalf("unknown messenger type: %s", typ)
	return nil
}
