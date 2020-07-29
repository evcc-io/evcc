package push

import (
	"fmt"
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
func NewMessengerFromConfig(typ string, other map[string]interface{}) (res Sender, err error) {
	switch strings.ToLower(typ) {
	case "pushover":
		var cc pushOverConfig
		if err = util.DecodeOther(other, &cc); err == nil {
			res = NewPushOverMessenger(cc.App, cc.Recipients)
		}
	case "telegram":
		var cc telegramConfig
		if err = util.DecodeOther(other, &cc); err == nil {
			res = NewTelegramMessenger(cc.Token, cc.Chats)
		}
	default:
		err = fmt.Errorf("unknown messenger type: %s", typ)
	}

	return res, err
}
