package push

import (
	"errors"
	"strings"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("email", NewEmailFromConfig)
	registry.Add("shout", NewShoutrrrFromConfig)
}

// Shoutrrr implements the shoutrrr messaging aggregator
type Shoutrrr struct {
	log *util.Logger
	app *router.ServiceRouter
}

// NewEmailFromConfig creates new email messenger based on Shoutrrr messenger
func NewEmailFromConfig(other map[string]any) (Messenger, error) {
	var cc struct {
		URI      string
		Host     string
		Port     string
		User     string
		Password string
		From     string
		To       []string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		if cc.Host == "" || cc.Port == "" || cc.User == "" || cc.Password == "" || cc.From == "" || len(cc.To) == 0 {
			return nil, errors.New("missing uri")
		}
		cc.URI = "smtp://" + cc.User + ":" + cc.Password + "@" + cc.Host + ":" + cc.Port + "/?fromAddress=" + cc.From + "&to=" + strings.Join(cc.To, ",")
	}

	return NewShoutrrrFromConfig(map[string]any{
		"uri": cc.URI,
	})
}

// NewShoutrrrFromConfig creates new Shoutrrr messenger
func NewShoutrrrFromConfig(other map[string]any) (Messenger, error) {
	var cc struct {
		URI string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	app, err := shoutrrr.CreateSender(cc.URI)
	if err != nil {
		return nil, err
	}

	m := &Shoutrrr{
		log: util.NewLogger("shoutrrr"),
		app: app,
	}

	return m, nil
}

// Send sends to all receivers
func (m *Shoutrrr) Send(title, msg string) {
	params := &types.Params{
		"title": title,
	}

	for _, err := range m.app.Send(msg, params) {
		if err != nil {
			m.log.ERROR.Println("send:", err)
		}
	}
}
