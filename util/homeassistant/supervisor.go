package homeassistant

import (
	"os"
	"strings"

	"golang.org/x/oauth2"
)

const (
	SupervisorURI      = "http://supervisor/core"
	SupervisorToken    = "SUPERVISOR_TOKEN"
	SupervisorInstance = "HomeAssistant via EVCC App"
)

func init() {
	if hasSupervisorToken() {
		addInstance(SupervisorInstance, SupervisorURI)
	}
}

func hasSupervisorToken() bool {
	return os.Getenv(SupervisorToken) != ""
}

func supervisorTokenSource(uri string) (oauth2.TokenSource, bool) {
	if token := os.Getenv(SupervisorToken); token != "" && strings.TrimRight(uri, "/") == SupervisorURI {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}), true
	}
	return nil, false
}
