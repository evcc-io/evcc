package core

import (
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
)

// isVehicleAtHome checks whether vehicle is at home (geofencing)
// false: if vehicle position is available and outside the defined radius
// true: in all other cases, even in cases of error or if position is not available
func (lp *Loadpoint) isVehicleAtHome(vehicle api.Vehicle) bool {
	geofence := lp.GetGeofenceConfig()

	if !geofence.Enabled || vehicle == nil {
		return true
	}

	vs, ok := vehicle.(api.VehiclePosition)
	if !ok {
		lp.log.DEBUG.Println("vehicle does not support position tracking")
		return true
	}

	lat, lon, err := vs.Position()
	if err != nil {
		lp.log.INFO.Printf("vehicle position: %v", err)
		return true
	}

	// {lat: 0, lon: 0} is a point in the atlantic ocean, not accessible by car
	// zero values indicate that the vehicle is not sending any valid position data
	if lat == 0 && lon == 0 {
		lp.log.INFO.Println("vehicle position: not available")
		return true
	}

	lp.log.DEBUG.Printf("vehicle position: lat %.4f, lon %.4f", lat, lon)

	atHome := isAtHome(geofence, lat, lon)

	lp.log.DEBUG.Printf(
		"vehicle distance from loadpoint: %.1fm (radius: %vm, atHome=%v)",
		distance(geofence.Lat, geofence.Lon, lat, lon)*1e3,
		geofence.Radius,
		atHome,
	)

	return atHome
}

// validate geofence settings
func validateGeofenceConfig(geofence loadpoint.GeofenceConfig) error {
	if geofence.Enabled {
		if (geofence.Lat == 0 && geofence.Lon == 0) ||
			geofence.Radius <= 0 ||
			math.Abs(geofence.Lat) > 90 ||
			math.Abs(geofence.Lon) > 180 {
			return fmt.Errorf("invalid geofence settings: %+v", geofence)
		}
	}
	return nil
}

// given a valid geofence and vehicle coords, is it at home?
func isAtHome(geofence loadpoint.GeofenceConfig, lat, lon float64) bool {
	d := distance(geofence.Lat, geofence.Lon, lat, lon) * 1e3
	return d <= geofence.Radius
}

// calculate the distance between two points on a globe
func distance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // km

	// Differences in radiant
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1 = lat1 * math.Pi / 180.0
	lat2 = lat2 * math.Pi / 180.0

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return c * earthRadius
}
