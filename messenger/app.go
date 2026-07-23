package messenger

import (
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	expoPushURI = "https://exp.host/--/api/v2/push/send"

	// Expo push tokens have the form ExponentPushToken[xxxxxxxx]
	tokenPrefix = "ExponentPushToken["
	tokenSuffix = "]"
	maxTokenLen = 128
	maxTokens   = 20
)

// AppPush sends messages to registered companion app devices via the Expo push
// service. The app registers its device token through the /api/push/token endpoint.
type AppPush struct {
	mu     sync.Mutex
	log    *util.Logger
	tokens []string
}

// NewAppPushFromSettings creates an AppPush messenger with tokens restored from settings
func NewAppPushFromSettings() *AppPush {
	m := &AppPush{log: util.NewLogger("apppush")}
	_ = settings.Json(keys.PushTokens, &m.tokens)
	return m
}

// ValidPushToken checks the Expo push token format
func ValidPushToken(token string) bool {
	return len(token) <= maxTokenLen &&
		strings.HasPrefix(token, tokenPrefix) &&
		strings.HasSuffix(token, tokenSuffix)
}

// Register adds a device token
func (m *AppPush) Register(token string) {
	if !ValidPushToken(token) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if slices.Contains(m.tokens, token) {
		return
	}

	// drop oldest when full
	if len(m.tokens) >= maxTokens {
		m.tokens = m.tokens[len(m.tokens)-maxTokens+1:]
	}

	m.tokens = append(m.tokens, token)
	m.persist()
}

// Unregister removes a device token
func (m *AppPush) Unregister(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if i := slices.Index(m.tokens, token); i >= 0 {
		m.tokens = slices.Delete(m.tokens, i, i+1)
		m.persist()
	}
}

// persist must be called with mu held
func (m *AppPush) persist() {
	if err := settings.SetJson(keys.PushTokens, m.tokens); err != nil {
		m.log.ERROR.Println(err)
	}
}

type expoPushMessage struct {
	To    string `json:"to"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body"`
}

type expoPushResponse struct {
	Data []struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Details struct {
			Error string `json:"error"`
		} `json:"details"`
	} `json:"data"`
}

// Send implements the api.Messenger interface
func (m *AppPush) Send(title, msg string) {
	m.mu.Lock()
	tokens := slices.Clone(m.tokens)
	m.mu.Unlock()

	if len(tokens) == 0 {
		return
	}

	messages := make([]expoPushMessage, 0, len(tokens))
	for _, to := range tokens {
		messages = append(messages, expoPushMessage{To: to, Title: title, Body: msg})
	}

	req, err := request.New(http.MethodPost, expoPushURI, request.MarshalJSON(messages), request.JSONEncoding)
	if err != nil {
		m.log.ERROR.Println(err)
		return
	}

	var res expoPushResponse
	if err := request.NewHelper(m.log).DoJSON(req, &res); err != nil {
		m.log.ERROR.Println(err)
		return
	}

	// responses are order-aligned with the request
	for i, r := range res.Data {
		if r.Status != "ok" && i < len(tokens) {
			m.log.WARN.Printf("push failed: %s %s", r.Message, r.Details.Error)

			// prune devices that are no longer registered
			if r.Details.Error == "DeviceNotRegistered" {
				m.Unregister(tokens[i])
			}
		}
	}
}
