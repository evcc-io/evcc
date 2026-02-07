package charger

import (
	"errors"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

// VehicleApi is a charger implementation that uses the vehicle api
// This is useful for "granny chargers" or simple chargers that can't be controlled directly
type VehicleApi struct {
	lp                     loadpoint.API
	enabled                bool
	geofenceEnabled        bool
	lat, lon, radius       float64
	cacheRefreshExpectedAt time.Time
}

func init() {
	registry.Add("vehicle-api", NewVehicleApiFromConfig)
}

// NewVehicleApiFromConfig creates a new vehicle-api charger
func NewVehicleApiFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		GeofenceEnabled bool    `mapstructure:"geofence_enabled"`
		Lat             float64 `mapstructure:"lat"`
		Lon             float64 `mapstructure:"lon"`
		Radius          float64 `mapstructure:"radius"`
	}{
		Radius: 100, // Default 100 meter radius
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	c := &VehicleApi{
		geofenceEnabled: cc.GeofenceEnabled,
		lat:             cc.Lat,
		lon:             cc.Lon,
		radius:          cc.Radius,
	}

	return c, nil
}

// isVehicleAtHome checks if the vehicle is within the geofence (if enabled)
func (c *VehicleApi) isVehicleAtHome(vehicle api.Vehicle) (bool, error) {
	if !c.geofenceEnabled {
		return true, nil // Assume at charger if geofencing is disabled
	}

	positioner, ok := vehicle.(api.VehiclePosition)
	if !ok {
		return false, errors.New("vehicle must support position tracking if geofence is enabled")
	}

	lat, lon, err := positioner.Position()
	if err != nil {
		return false, err
	}

	return c.distance(lat, lon) <= c.radius, nil
}

// Status implements the api.Charger interface
func (c *VehicleApi) Status() (api.ChargeStatus, error) {
	if c.lp == nil {
		return api.StatusA, nil
	}

	vehicle := c.lp.GetVehicle()
	if vehicle == nil {
		return api.StatusA, nil // No vehicle = disconnected
	}

	// Check if vehicle is at the charger (trying to use geofencing)
	atHome, err := c.isVehicleAtHome(vehicle)
	if err != nil {
		return api.StatusA, err
	}

	if !c.cacheRefreshExpectedAt.IsZero() {
		if time.Now().Before(c.cacheRefreshExpectedAt) {
			if !c.enabled {
				// to avoid charge logic errors while waiting for cache refresh
				return api.StatusB, nil
			}
		} else {
			util.ResetCached()
			c.cacheRefreshExpectedAt = time.Time{}
		}
	}

	chargeState, ok := vehicle.(api.ChargeState)
	if !ok {
		return api.StatusA, errors.New("vehicle not capable of reporting charging status")
	}

	status, err := chargeState.Status()
	if err != nil {
		return api.StatusNone, err
	}

	if status == api.StatusA || !atHome {
		return api.StatusA, nil
	}

	return status, nil
}

// Enabled implements the api.Charger interface
func (c *VehicleApi) Enabled() (bool, error) {
	return verifyEnabled(c, c.enabled)
}

// Enable implements the api.Charger interface
func (c *VehicleApi) Enable(enable bool) error {
	if c.lp == nil {
		return errors.New("loadpoint not initialized")
	}

	status, err := c.Status()
	if err != nil {
		return err
	}

	// ignore disabling when vehicle is already disconnected
	if status == api.StatusA && !enable {
		c.enabled = false
		return nil
	}

	chargeController, ok := c.lp.GetVehicle().(api.ChargeController)
	if !ok {
		return errors.New("vehicle not capable of start/stop")
	}

	if err := chargeController.ChargeEnable(enable); err != nil {
		return err
	}

	c.enabled = enable
	// delayed reset if vehicle cache- allows vehicle APIs to reflect new charging status
	c.cacheRefreshExpectedAt = time.Now().Add(3 * time.Minute)

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *VehicleApi) MaxCurrent(current int64) error {
	if c.lp == nil {
		return errors.New("loadpoint not initialized")
	}

	currentController, ok := c.lp.GetVehicle().(api.CurrentController)
	if !ok {
		// If we cannot control the current, we just pretend that we do
		return nil
	}

	return currentController.MaxCurrent(current)
}

var _ loadpoint.Controller = (*VehicleApi)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *VehicleApi) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}

// distance approximates Euclidean distance, good enough for geofencing
func (c *VehicleApi) distance(lat, lon float64) float64 {
	const metersPerDegreeLat = 111000 // ~111km per degree lat (constant)

	deltaLat := (c.lat - lat) * metersPerDegreeLat
	deltaLon := (c.lon - lon) * metersPerDegreeLat * math.Cos(c.lat*math.Pi/180) // varies by lat

	return math.Sqrt(deltaLat*deltaLat + deltaLon*deltaLon)
}
