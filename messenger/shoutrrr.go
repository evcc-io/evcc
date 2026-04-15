package messenger

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/nicholas-fedor/shoutrrr"
	"github.com/nicholas-fedor/shoutrrr/pkg/router"
	"github.com/nicholas-fedor/shoutrrr/pkg/types"
)

func init() {
	registry.Add("shout", NewShoutrrrFromConfig)
}

// Shoutrrr implements the shoutrrr messaging aggregator
type Shoutrrr struct {
	log *util.Logger
	app *router.ServiceRouter
}

// NewShoutrrrFromConfig creates new Shoutrrr messenger
func NewShoutrrrFromConfig(other map[string]any) (api.Messenger, error) {
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
