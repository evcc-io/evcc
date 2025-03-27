package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("sax", NewSAXAuthFromConfig)
}

// NewSAXAuthFromConfig creates a SAX authentication plugin
func NewSAXAuthFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
	var cc struct {
		User     string
		Password string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, errors.New("missing user or password for SAX authentication")
	}

	return &SAXAuthPlugin{
		ctx:      ctx,
		log:      util.NewLogger("sax"),
		user:     cc.User,
		password: cc.Password,
		client:   &http.Client{Timeout: 10 * time.Second},
	}, nil
}

type SAXAuthPlugin struct {
	mu       sync.Mutex
	ctx      context.Context
	log      *util.Logger
	user     string
	password string
	token    string
	expires  time.Time
	client   *http.Client
}

// Token fetches or refreshes the authentication token
func (s *SAXAuthPlugin) Token() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return the cached token if it's still valid
	if time.Now().Before(s.expires) {
		return s.token, nil
	}

	// Fetch a new token
	url := "https://webserver.sax-power.net/api/auth/token/"
	payload := map[string]string{
		"email":    s.user,
		"password": s.password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch token: status %d", resp.StatusCode)
	}

	var res struct {
		Token   string `json:"access"`
		Expires int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	s.token = res.Token
	s.expires = time.Now().Add(time.Duration(res.Expires) * time.Second)
	return s.token, nil
}
