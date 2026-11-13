package messenger

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
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

	if u.Host == "ntfy.sh" {
		log = log.Redact(strings.TrimPrefix(u.Path, "/"))
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
		m.log.ERROR.Printf("%v", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		m.log.ERROR.Printf("%v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// the body carries the server's reason, the uri is left out as it holds the token
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		m.log.ERROR.Printf("unexpected status: %d (%s) %s", resp.StatusCode, http.StatusText(resp.StatusCode), strings.TrimSpace(string(body)))
	}
}
