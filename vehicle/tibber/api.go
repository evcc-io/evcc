package tibber

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/spf13/cast"
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

// VIN returns the vehicle identification number from the external id, which is
// formatted as vendor:vin (e.g. tesla:5YJSA1E26MF1234567).
func (d Device) VIN() string {
	if _, vin, ok := strings.Cut(d.ExternalID, ":"); ok {
		return vin
	}
	return d.ExternalID
}

// DeviceDetail is the full device state including capabilities.
type DeviceDetail struct {
	ID           string
	ExternalID   string
	Info         DeviceInfo
	Capabilities []Capability
}

// Capability is a single device capability and its last-seen value. The value
// is delivered as a JSON number or string.
type Capability struct {
	ID          string
	Description string
	Value       any
	Unit        string
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
	err := v.GetJSON(fmt.Sprintf("%s/homes", ApiURI), &res)
	return res.Homes, err
}

// Devices lists the devices of a home.
func (v *API) Devices(homeID string) ([]Device, error) {
	var res struct {
		Devices []Device
	}
	err := v.GetJSON(fmt.Sprintf("%s/homes/%s/devices", ApiURI, homeID), &res)
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
	err := v.GetJSON(fmt.Sprintf("%s/homes/%s/devices/%s", ApiURI, homeID, deviceID), &res)
	return res, err
}

// Tibber Data API capability ids and the enum values they report.
const (
	idSoc       = "storage.stateOfCharge"       // %
	idTargetSoc = "storage.targetStateOfCharge" // %
	idRange     = "range.remaining"             // distance, typically m
	idConnector = "connector.status"            // connected/disconnected/unknown
	idCharging  = "charging.status"             // charging/idle/unknown

	StatusConnected = "connected" // connector.status
	StatusCharging  = "charging"  // charging.status
)

const kmPerMile = 1.609344

// capability returns the capability with the given id.
func (d DeviceDetail) capability(id string) (Capability, bool) {
	for _, c := range d.Capabilities {
		if c.ID == id {
			return c, true
		}
	}
	return Capability{}, false
}

// Soc returns the battery state of charge in percent.
func (d DeviceDetail) Soc() (float64, bool) {
	c, ok := d.capability(idSoc)
	if !ok {
		return 0, false
	}
	f, err := cast.ToFloat64E(c.Value)
	return f, err == nil
}

// TargetSoc returns the configured charge limit in percent.
func (d DeviceDetail) TargetSoc() (float64, bool) {
	c, ok := d.capability(idTargetSoc)
	if !ok {
		return 0, false
	}
	f, err := cast.ToFloat64E(c.Value)
	return f, err == nil
}

// Range returns the estimated range in km, converting m/mi to km as needed.
func (d DeviceDetail) Range() (float64, bool) {
	c, ok := d.capability(idRange)
	if !ok {
		return 0, false
	}
	f, err := cast.ToFloat64E(c.Value)
	if err != nil {
		return 0, false
	}
	switch c.Unit {
	case "m":
		f /= 1000
	case "mi", "mile", "miles":
		f *= kmPerMile
	}
	return f, true
}

// PlugStatus returns the connector (plug) status value, e.g. connected,
// disconnected or unknown.
func (d DeviceDetail) PlugStatus() (string, bool) {
	return d.statusValue(idConnector)
}

// ChargingStatus returns the charging status value, e.g. charging, idle or unknown.
func (d DeviceDetail) ChargingStatus() (string, bool) {
	return d.statusValue(idCharging)
}

// statusValue returns the string value of the capability with the given id.
func (d DeviceDetail) statusValue(id string) (string, bool) {
	c, ok := d.capability(id)
	if !ok {
		return "", false
	}
	s, ok := c.Value.(string)
	return s, ok
}
