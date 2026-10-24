package ecoflow

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// powerOceanPrefixes are the PowerOcean serial prefixes.
// Source: https://github.com/Xygen/EcoFlow-PowerPulse-2-for-Home-Assistant (device_types.py)
var powerOceanPrefixes = []string{
	"HJ31", "HJ32", "HJ35", "HJ37", "HJ3C",
	"J327", "J32B", "J32D", "J32E",
	"R371", "R372", "R374",
}

func isPowerOcean(serial string) bool {
	return slices.ContainsFunc(powerOceanPrefixes, func(prefix string) bool {
		return strings.HasPrefix(serial, prefix)
	})
}

// Gateway returns the serial of the only PowerOcean in the account. It fails
// when the account has none or several.
func Gateway(log *util.Logger, session *Session) (string, error) {
	req, err := request.New(http.MethodGet, session.Base+"/iot-service/user/device", nil,
		request.AcceptJSON, map[string]string{"Authorization": "Bearer " + session.Token})
	if err != nil {
		return "", err
	}

	var res struct {
		response
		Data any `json:"data"`
	}
	if err := request.NewHelper(log).DoJSON(req, &res); err != nil {
		return "", fmt.Errorf("device list: %w", err)
	}
	if err := res.error(); err != nil {
		return "", fmt.Errorf("device list: %w", err)
	}

	devices := slices.Compact(slices.Sorted(slices.Values(serials(res.Data))))
	gateways := slices.DeleteFunc(slices.Clone(devices), func(serial string) bool { return !isPowerOcean(serial) })

	switch len(gateways) {
	case 0:
		return "", fmt.Errorf("no PowerOcean found in the account (devices: %s), configure gateway", strings.Join(devices, ", "))
	case 1:
		return gateways[0], nil
	default:
		return "", fmt.Errorf("several PowerOcean found in the account (%s), configure gateway", strings.Join(gateways, ", "))
	}
}

// serials collects the device serials of the device list. The groups (bound,
// share) hold either lists of records with an sn field or maps keyed by serial.
func serials(v any) []string {
	var res []string

	switch v := v.(type) {
	case []any:
		for _, e := range v {
			res = append(res, serials(e)...)
		}
	case map[string]any:
		for k, e := range v {
			if k == "sn" {
				if sn, ok := e.(string); ok {
					res = append(res, strings.ToUpper(sn))
				}
				continue
			}
			if key := strings.ToUpper(k); isPowerOcean(key) {
				res = append(res, key)
			}
			res = append(res, serials(e)...)
		}
	}

	return res
}

// Gateway returns the serial of the only PowerOcean in the account of the
// connection's login.
func (c *Connection) Gateway() (string, error) {
	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	if session == nil {
		return "", errors.New("no login to read the device list")
	}

	return Gateway(c.log, session)
}
