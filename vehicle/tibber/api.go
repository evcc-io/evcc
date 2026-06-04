package tibber

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// Home is an entry of the customer's home list.
type Home struct {
	ID   string
	Name string
}

// Device is a device list entry (identity only, no live values).
type Device struct {
	ID         string
	ExternalID string
	Info       DeviceInfo
}

// DeviceInfo holds static make/brand/model information.
type DeviceInfo struct {
	Name  string
	Brand string
	Model string
}

// DeviceDetail is the full device state including capabilities.
type DeviceDetail struct {
	ID           string
	ExternalID   string
	Info         DeviceInfo
	Capabilities []Capability
}

// Capability is a single device capability and its last-seen value. The value
// is delivered as a JSON number or string; Float normalizes both.
type Capability struct {
	ID          string
	Description string
	Value       any
	Unit        string
}

// Float returns the capability value as a float, accepting both number and string encodings.
func (c Capability) Float() (float64, error) {
	switch v := c.Value.(type) {
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unexpected value type %T for capability %s", c.Value, c.ID)
	}
}

// API is the Tibber Data API REST client.
type API struct {
	*request.Helper
}

// NewAPI creates a Tibber Data API client authenticated via the given token source.
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	client := request.NewHelper(log)
	client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   client.Transport,
	}
	return &API{Helper: client}
}

// Homes lists the homes the user has access to.
func (v *API) Homes() ([]Home, error) {
	var res struct {
		Homes []Home
	}
	err := v.GetJSON(fmt.Sprintf("%s/homes", URI), &res)
	return res.Homes, err
}

// Devices lists the devices of a home.
func (v *API) Devices(homeID string) ([]Device, error) {
	var res struct {
		Devices []Device
	}
	err := v.GetJSON(fmt.Sprintf("%s/homes/%s/devices", URI, homeID), &res)
	return res.Devices, err
}

// Vehicles lists the vehicles across all homes, deduplicated. With only the
// vehicles scope granted, the devices endpoint returns vehicles only.
func (v *API) Vehicles() ([]Device, error) {
	homes, err := v.Homes()
	if err != nil {
		return nil, err
	}

	var res []Device
	seen := make(map[string]bool)

	for _, home := range homes {
		devices, err := v.Devices(home.ID)
		if err != nil {
			return nil, err
		}

		for _, d := range devices {
			if !seen[d.ID] {
				seen[d.ID] = true
				res = append(res, d)
			}
		}
	}

	return res, nil
}

// Device returns the full state of a single device.
func (v *API) Device(homeID, deviceID string) (DeviceDetail, error) {
	var res DeviceDetail
	uri := fmt.Sprintf("%s/homes/%s/devices/%s", URI, homeID, deviceID)
	req, err := request.New(http.MethodGet, uri, nil, request.AcceptJSON)
	if err != nil {
		return res, err
	}
	err = v.DoJSON(req, &res)
	return res, err
}

// capability returns the first capability matching the predicate.
func (d DeviceDetail) capability(match func(Capability) bool) (Capability, bool) {
	for _, c := range d.Capabilities {
		if match(c) {
			return c, true
		}
	}
	return Capability{}, false
}

// Soc returns the battery state of charge in percent. It matches the capability
// reporting a percentage value (the Tibber capability ids are not contractual).
func (d DeviceDetail) Soc() (float64, bool) {
	c, ok := d.capability(func(c Capability) bool {
		return c.Unit == "%" && hint(c, "charge", "soc", "battery")
	})
	if !ok {
		return 0, false
	}
	f, err := c.Float()
	return f, err == nil
}

// Range returns the estimated range in km. It matches the capability reporting
// a distance value.
func (d DeviceDetail) Range() (float64, bool) {
	c, ok := d.capability(func(c Capability) bool {
		return (c.Unit == "km" || c.Unit == "mi") && hint(c, "range")
	})
	if !ok {
		return 0, false
	}
	f, err := c.Float()
	return f, err == nil
}

// hint reports whether the capability id or description contains any of the keywords.
func hint(c Capability, keywords ...string) bool {
	hay := strings.ToLower(c.ID + " " + c.Description)
	for _, k := range keywords {
		if strings.Contains(hay, k) {
			return true
		}
	}
	return false
}
