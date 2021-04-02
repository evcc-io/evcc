package tasmota

import (
	"fmt"
	"net/url"
	"strings"
)

// CreateRequest creates the Tasmota command web request
// https://tasmota.github.io/docs/Commands/#with-web-requests
func CreateRequest(uri, user, password, cmnd string) string {
	parameters := url.Values{
		"user":     []string{user},
		"password": []string{password},
		"cmnd":     []string{cmnd},
	}

	return fmt.Sprintf("%s/cm?%s", strings.TrimRight(uri, "/"), parameters.Encode())
}
