package homeassistant

import (
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/api"
)

// GetClimateState retrieves the state and temperature attributes of a climate entity.
func (c *Connection) GetClimateState(entity string) (ClimateStateResponse, error) {
	var res ClimateStateResponse
	uri := fmt.Sprintf("%s/api/states/%s", c.instance.URI(), url.PathEscape(entity))
	if err := c.GetJSON(uri, &res); err != nil {
		return res, err
	}
	if res.State == "unknown" || res.State == "unavailable" {
		return res, api.ErrNotAvailable
	}
	return res, nil
}
