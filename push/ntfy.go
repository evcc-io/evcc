package push

import (
	"errors"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("ntfy", NewNtfyFromConfig)
}

// Ntfy implements the ntfy messaging aggregator
type Ntfy struct {
	log      *util.Logger
	uri      string
	priority string
	tags     string
}

// NewNtfyFromConfig creates new Ntfy messenger
func NewNtfyFromConfig(other map[string]interface{}) (Messenger, error) {
	var cc struct {
		URI      string
		Priority string
		Tags     string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	m := &Ntfy{
		log:      util.NewLogger("ntfy"),
		uri:      cc.URI,
		priority: cc.Priority,
		tags:     cc.Tags,
	}

	return m, nil
}

// Send sends to all receivers
func (m *Ntfy) Send(title, msg string) {
	req, err := request.New("POST", m.uri, strings.NewReader(msg), map[string]string{
		"Priority": m.priority,
		"Title":    title,
		"Tags":     m.tags,
	})
	if err != nil {
		m.log.ERROR.Printf("ntfy: %v", err)
	}

	if _, err := http.DefaultClient.Do(req); err != nil {
		m.log.ERROR.Printf("ntfy: %v", err)
	}
}
