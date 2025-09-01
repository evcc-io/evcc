package charger

import (
	"errors"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

// VehicleControlledCharger is a charger implementation that delegates control to the vehicle
// This is useful for "granny chargers" or simple chargers that can't be controlled directly
type VehicleControlledCharger struct {
	log              *util.Logger
	lp               loadpoint.API
	enabled          bool
	isCharging       bool
	wasDisconnected  bool
	geofenceEnabled  bool
	chargerLatitude  float64
	chargerLongitude float64
	radius           float64
}

func init() {
	registry.Add("vehicle-controlled", NewVehicleControlledChargerFromConfig)
}

// NewVehicleControlledChargerFromConfig creates a new vehicle-controlled charger
func NewVehicleControlledChargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		GeofenceEnabled bool    `mapstructure:"geofence_enabled"`
		Latitude        float64 `mapstructure:"latitude"`
		Longitude       float64 `mapstructure:"longitude"`
		Radius          float64 `mapstructure:"radius"`
	}{
		Radius: 100, // Default 100 meter radius
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	c := &VehicleControlledCharger{
		log:              util.NewLogger("vehicle-controlled"),
		enabled:          false,
		isCharging:       false,
		wasDisconnected:  true,
		geofenceEnabled:  cc.GeofenceEnabled,
		chargerLatitude:  cc.Latitude,
		chargerLongitude: cc.Longitude,
		radius:           cc.Radius,
	}

	return c, nil
}

// isVehicleAtCharger checks if the vehicle is within the geofence (if enabled)
func (c *VehicleControlledCharger) isVehicleAtCharger(vehicle api.Vehicle) (bool, error) {
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

	distance := simpleDistance(c.chargerLatitude, c.chargerLongitude, lat, lon)
	return distance <= c.radius, nil
}

// Status implements the api.Charger interface
func (c *VehicleControlledCharger) Status() (api.ChargeStatus, error) {
	if c.lp == nil {
		return api.StatusA, nil
	}

	vehicle := c.lp.GetVehicle()
	if vehicle == nil {
		return api.StatusA, nil // No vehicle = disconnected
	}

	// Check if vehicle is at the charger (trying to use geofencing)
	vehicleIsAtCharger, err := c.isVehicleAtCharger(vehicle)
	if err != nil {
		return api.StatusA, err
	}

	chargeState, ok := vehicle.(api.ChargeState)
	if !ok {
		return api.StatusA, errors.New("vehicle not capable of reporting charging status")
	}

	vehicleAPIStatus, err := chargeState.Status()
	if err != nil {
		return api.StatusNone, err
	}

	if vehicleAPIStatus == api.StatusA || !vehicleIsAtCharger {
		c.wasDisconnected = true
		c.isCharging = false
		return api.StatusA, nil
	}

	if c.wasDisconnected {
		c.isCharging = vehicleAPIStatus == api.StatusC
		c.wasDisconnected = false
	}

	if c.isCharging {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (c *VehicleControlledCharger) Enabled() (bool, error) {
	return verifyEnabled(c, c.enabled)
}

// Enable implements the api.Charger interface
func (c *VehicleControlledCharger) Enable(enable bool) error {
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
		c.isCharging = false
		return nil
	}

	chargeController, ok := c.lp.GetVehicle().(api.ChargeController)
	if !ok {
		return errors.New("vehicle not capable of start/stop")
	}

	err = chargeController.ChargeEnable(enable)
	if err != nil {
		return err
	}
	c.enabled = enable
	c.isCharging = enable

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *VehicleControlledCharger) MaxCurrent(current int64) error {
	if c.lp == nil {
		return errors.New("loadpoint not initialized")
	}

	currentController, ok := c.lp.GetVehicle().(api.CurrentController)
	if !ok {
		return nil
		// If we cannot control the current, we just pretend that we do
	}

	return currentController.MaxCurrent(current)
}

var _ loadpoint.Controller = (*VehicleControlledCharger)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *VehicleControlledCharger) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}

func simpleDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Approximate Eucledian distance, good enough for geofencing
	const metersPerDegreeLat = 111000                         // ~111km per degree latitude (constant)
	metersPerDegreeLng := 111000 * math.Cos(lat1*math.Pi/180) // varies by latitude

	deltaLat := (lat2 - lat1) * metersPerDegreeLat
	deltaLng := (lng2 - lng1) * metersPerDegreeLng

	return math.Sqrt(deltaLat*deltaLat + deltaLng*deltaLng)
}
