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
	log             *util.Logger
	lp              loadpoint.API
	enabled         bool
	geofenceEnabled bool
	homeLatitude    float64
	homeLongitude   float64
	radiusMeters    float64
}

func init() {
	registry.Add("vehicle-controlled", NewVehicleControlledChargerFromConfig)
}

// NewVehicleControlledChargerFromConfig creates a new vehicle-controlled charger
func NewVehicleControlledChargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		GeofenceEnabled bool    `mapstructure:"geofence_enabled"`
		HomeLatitude    float64 `mapstructure:"home_latitude"`
		HomeLongitude   float64 `mapstructure:"home_longitude"`
		RadiusMeters    float64 `mapstructure:"radius_meters"`
	}{
		GeofenceEnabled: false,
		RadiusMeters:    100, // Default 100 meter radius
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	c := &VehicleControlledCharger{
		log:             util.NewLogger("vehicle-controlled"),
		geofenceEnabled: cc.GeofenceEnabled,
		homeLatitude:    cc.HomeLatitude,
		homeLongitude:   cc.HomeLongitude,
		radiusMeters:    cc.RadiusMeters,
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

	lat, lng, err := positioner.Position()
	if err != nil {
		return true, errors.New("vehicle must support position tracking if geofence is enabled")
	}

	distance := c.simpleDistance(c.homeLatitude, c.homeLongitude, lat, lng)
	return distance <= c.radiusMeters, nil
}

func (c *VehicleControlledCharger) simpleDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Approximate Eucledian distance, good enough for geofencing
	const metersPerDegreeLat = 111000                         // ~111km per degree latitude (constant)
	metersPerDegreeLng := 111000 * math.Cos(lat1*math.Pi/180) // varies by latitude

	deltaLat := (lat2 - lat1) * metersPerDegreeLat
	deltaLng := (lng2 - lng1) * metersPerDegreeLng

	return math.Sqrt(deltaLat*deltaLat + deltaLng*deltaLng)
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
		c.log.ERROR.Printf("Error checking if vehicle is at charger: %v", err)
		return api.StatusA, err
	}
	if !vehicleIsAtCharger {
		return api.StatusA, nil // Vehicle not at charger = disconnected
	}

	// If the vehicle is at the charger, then the vehicles charge state is the charge state of this charger
	chargeState, ok := vehicle.(api.ChargeState)
	if !ok {
		return api.StatusA, errors.New("vehicle must support charge state if using vehicle-controlled charger")
	}

	return chargeState.Status()
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
		return nil
	}

	vehicle := c.lp.GetVehicle()
	if vehicle == nil {
		return errors.New("no vehicle configured")
	}

	chargeController, ok := vehicle.(api.ChargeController)
	if !ok {
		return errors.New("vehicle not capable of controlling charging")
	}

	err = chargeController.ChargeEnable(enable)
	if err == nil {
		c.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *VehicleControlledCharger) MaxCurrent(current int64) error {
	if c.lp == nil {
		return errors.New("loadpoint not initialized")
	}

	vehicle := c.lp.GetVehicle()
	if vehicle == nil {
		return errors.New("no vehicle configured")
	}

	currentController, ok := vehicle.(api.CurrentController)
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
