package push

import (
	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("email", NewShoutrrrFromConfig)
	registry.Add("shout", NewShoutrrrFromConfig)
}

// Shoutrrr implements the shoutrrr messaging aggregator
type Shoutrrr struct {
	log *util.Logger
	app *router.ServiceRouter
}

// NewShoutrrrFromConfig creates new Shoutrrr messenger
func NewShoutrrrFromConfig(other map[string]interface{}) (Messenger, error) {
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
