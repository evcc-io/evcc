package messenger

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/api"
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
func NewNtfyFromConfig(other map[string]any) (api.Messenger, error) {
	var cc struct {
		URI       string
		Priority  string
		Tags      string
		AuthToken string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	u, err := url.Parse(cc.URI)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("ntfy")

	if cc.AuthToken != "" {
		bearer := "Bearer " + cc.AuthToken
		encoded := base64.RawStdEncoding.EncodeToString([]byte(bearer))

		q := u.Query()
		if q.Has("auth") {
			return nil, fmt.Errorf("uri already contains auth parameter")
		}

		q.Set("auth", encoded)
		u.RawQuery = q.Encode()

		cc.URI = u.String()

		log = log.Redact(cc.AuthToken, bearer, encoded)
	}

	if token, ok := strings.CutPrefix(u.String(), "https://ntfy.sh/"); ok {
		log = log.Redact(token)
	}

	m := &Ntfy{
		log:      log,
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
