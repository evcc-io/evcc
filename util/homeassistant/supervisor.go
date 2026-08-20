package homeassistant

import (
	"os"
	"strings"

	"golang.org/x/oauth2"
)

const (
	// SupervisorURI is the Home Assistant Core API endpoint when running as a Home Assistant add-on
	SupervisorURI = "http://supervisor/core"
	// SupervisorToken is the environment variable name containing the bearer token
	SupervisorToken = "SUPERVISOR_TOKEN"
	// SupervisorInstance is the discovered instance name for the Supervisor integration
	SupervisorInstance = "HomeAssistant Host"
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
