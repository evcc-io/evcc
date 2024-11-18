package push

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates Messenger from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]interface{}) (Messenger, error) {
	var cc struct {
		Send     provider.Config
		Encoding string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	send, err := provider.NewStringSetterFromConfig(ctx, "send", cc.Send)
	if err != nil {
		return nil, err
	}

	return NewConfigurable(send, cc.Encoding)
}

// NewConfigurable creates a new Messenger
func NewConfigurable(send func(string) error, encoding string) (*Push, error) {
	m := &Push{
		log:      util.NewLogger("push"),
		send:     send,
		encoding: strings.ToLower(encoding),
	}
	return m, nil
}

// Push is a configurable Messenger implementation
type Push struct {
	log      *util.Logger
	send     func(string) error
	encoding string
}

func (m *Push) csv(separator rune, title, msg string) string {
	var b bytes.Buffer
	ww := csv.NewWriter(&b)
	ww.Comma = separator
	_ = ww.Write([]string{title, msg})
	ww.Flush()
	return b.String()
}

// Send implements the Messenger interface
func (m *Push) Send(title, msg string) {
	var res string

	switch m.encoding {
	case "json":
		b, _ := json.Marshal(struct {
			Title string `json:"title,omitempty"`
			Msg   string `json:"msg"`
		}{
			Title: title,
			Msg:   msg,
		})
		res = string(b)
	case "csv":
		res = m.csv(',', title, msg)
	case "tsv":
		res = m.csv('\t', title, msg)
	case "title":
		res = title
	default:
		res = msg
	}

	if err := m.send(res); err != nil {
		m.log.ERROR.Printf("send: %v", err)
	}
}
