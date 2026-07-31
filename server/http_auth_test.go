package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/auth"
	"github.com/stretchr/testify/assert"
)

// fakeAuth is a minimal auth.Auth stub for gate tests.
type fakeAuth struct {
	mode     auth.AuthMode
	password string
	apiKey   string
}

func (f fakeAuth) GetAuthMode() auth.AuthMode                     { return f.mode }
func (f fakeAuth) IsAdminPasswordValid(pw string) bool            { return pw != "" && pw == f.password }
func (f fakeAuth) ValidateApiKey(key string) bool                 { return key != "" && key == f.apiKey }
func (f fakeAuth) SetAuthMode(auth.AuthMode)                      {}
func (f fakeAuth) RemoveAdminPassword()                           {}
func (f fakeAuth) SetAdminPassword(string) error                  { return nil }
func (f fakeAuth) GenerateJwtToken(time.Duration) (string, error) { return "", nil }
func (f fakeAuth) ValidateJwtToken(string) bool                   { return true }
func (f fakeAuth) IsAdminPasswordConfigured() bool                { return f.password != "" }
func (f fakeAuth) SetApiKey() (string, error)                     { return "", nil }
func (f fakeAuth) IsApiKeyConfigured() bool                       { return f.apiKey != "" }

func TestRequireCriticalConfig(t *testing.T) {
	const pw = "secret"
	const key = "evcc_token"
	scriptReq := configReq{Yaml: "power:\n  source: script\n  cmd: echo 1"}
	scriptListReq := configReq{Yaml: "- name: main\n  getmaxcurrent:\n    source: script\n    cmd: echo 1"}
	benignReq := configReq{Yaml: "power:\n  source: http\n  uri: http://localhost"}

	base := fakeAuth{mode: auth.Enabled, password: pw, apiKey: key}

	tc := []struct {
		name   string
		req    configReq
		header map[string]string
		mode   auth.AuthMode
		ok     bool
	}{
		{"no critical plugin passes", benignReq, nil, auth.Enabled, true},
		{"disabled mode passes", scriptReq, nil, auth.Disabled, true},
		{"session without password rejected", scriptReq, nil, auth.Enabled, false},
		{"session with wrong password rejected", scriptReq, map[string]string{"X-Admin-Password": "nope"}, auth.Enabled, false},
		{"session with valid password passes", scriptReq, map[string]string{"X-Admin-Password": pw}, auth.Enabled, true},
		{"valid api key passes without password", scriptReq, map[string]string{"Authorization": "Bearer " + key}, auth.Enabled, true},
		{"yaml list with script rejected without password", scriptListReq, nil, auth.Enabled, false},
		{"yaml list with script passes with password", scriptListReq, map[string]string{"X-Admin-Password": pw}, auth.Enabled, true},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			a := base
			a.mode = tc.mode

			r := httptest.NewRequest(http.MethodPost, "/api/config/test/meter", nil)
			for k, v := range tc.header {
				r.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			ok := requireCriticalConfigAuth(w, r, a, tc.req)

			assert.Equal(t, tc.ok, ok)
			if !tc.ok {
				assert.Equal(t, http.StatusPreconditionRequired, w.Code)
			}
		})
	}
}
