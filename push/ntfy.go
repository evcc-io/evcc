package push

import (
	"errors"
	"net/http"
	"strings"
)

// Ntfy implements the ntfy messaging aggregator
type Ntfy struct {
	uri string
	priority string
	tags string
}

type ntfyConfig struct {
	URI string
	Priority string
	Tags string
}

// NewNtfyMessenger creates new Ntfy messenger
func NewNtfyMessenger(uri string, priority string, tags string) (*Ntfy, error) {
	if uri == "" {
	    return nil, errors.New("ntfy: missing uri")
	}

	m := &Ntfy{
		uri: uri,
		priority: priority,
		tags: tags,
	}

	return m, nil
}

// Send sends to all receivers
func (m *Ntfy) Send(title, msg string) {
	req, err := http.NewRequest("POST", m.uri, strings.NewReader(msg))
	if err != nil {
		log.ERROR.Printf("ntfy: %v", err)
	}
	req.Header.Set("Title", title)
	req.Header.Set("Priority", m.priority)
	req.Header.Set("Tags", m.tags)
	if _, err := http.DefaultClient.Do(req); err != nil {
		log.ERROR.Printf("ntfy: %v", err)
	}
}
