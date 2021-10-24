package push

import (
	"fmt"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
)

// Shoutrrr implements the shoutrrr messaging aggregator
type Shoutrrr struct {
	app *router.ServiceRouter
}

type shoutrrrConfig struct {
	URI string
}

// NewShoutrrrMessenger creates new Shoutrrr messenger
func NewShoutrrrMessenger(uri string) (*Shoutrrr, error) {
	app, err := shoutrrr.CreateSender(uri)
	if err != nil {
		return nil, fmt.Errorf("shoutrrr: %v", err)
	}

	m := &Shoutrrr{
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
			log.ERROR.Printf("shoutrrr: %v", err)
		}
	}
}
