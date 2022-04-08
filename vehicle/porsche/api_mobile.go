package porsche

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util/log"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const (
	MobileApiURI = "https://api.ppa.porsche.com"

	// vehicle status elements
	ACV_STATE                     = "ACV_STATE"
	BATTERY_CHARGING_STATE        = "BATTERY_CHARGING_STATE"
	BATTERY_LEVEL                 = "BATTERY_LEVEL"
	BATTERY_TYPE                  = "BATTERY_TYPE"
	BLEID_DDADATA                 = "BLEID_DDADATA"
	CAR_ALARMS_HISTORY            = "CAR_ALARMS_HISTORY"
	CHARGING_PROFILES             = "CHARGING_PROFILES"
	CLIMATIZER_STATE              = "CLIMATIZER_STATE"
	E_RANGE                       = "E_RANGE"
	FUEL_LEVEL                    = "FUEL_LEVEL"
	FUEL_RESERVE                  = "FUEL_RESERVE"
	GLOBAL_PRIVACY_MODE           = "GLOBAL_PRIVACY_MODE"
	GPS_LOCATION                  = "GPS_LOCATION"
	HEATING_STATE                 = "HEATING_STATE"
	INTERMEDIATE_SERVICE_RANGE    = "INTERMEDIATE_SERVICE_RANGE"
	INTERMEDIATE_SERVICE_TIME     = "INTERMEDIATE_SERVICE_TIME"
	LOCATION_ALARMS               = "LOCATION_ALARMS"
	LOCATION_ALARMS_HISTORY       = "LOCATION_ALARMS_HISTORY"
	LOCK_STATE_VEHICLE            = "LOCK_STATE_VEHICLE"
	MAIN_SERVICE_RANGE            = "MAIN_SERVICE_RANGE"
	MAIN_SERVICE_TIME             = "MAIN_SERVICE_TIME"
	MILEAGE                       = "MILEAGE"
	OIL_LEVEL_CURRENT             = "OIL_LEVEL_CURRENT"
	OIL_LEVEL_MAX                 = "OIL_LEVEL_MAX"
	OIL_LEVEL_MIN_WARNING         = "OIL_LEVEL_MIN_WARNING"
	OIL_SERVICE_RANGE             = "OIL_SERVICE_RANGE"
	OIL_SERVICE_TIME              = "OIL_SERVICE_TIME"
	OPEN_STATE_CHARGE_FLAP_LEFT   = "OPEN_STATE_CHARGE_FLAP_LEFT"
	OPEN_STATE_CHARGE_FLAP_RIGHT  = "OPEN_STATE_CHARGE_FLAP_RIGHT"
	OPEN_STATE_DOOR_FRONT_LEFT    = "OPEN_STATE_DOOR_FRONT_LEFT"
	OPEN_STATE_DOOR_FRONT_RIGHT   = "OPEN_STATE_DOOR_FRONT_RIGHT"
	OPEN_STATE_DOOR_REAR_LEFT     = "OPEN_STATE_DOOR_REAR_LEFT"
	OPEN_STATE_DOOR_REAR_RIGHT    = "OPEN_STATE_DOOR_REAR_RIGHT"
	OPEN_STATE_LID_FRONT          = "OPEN_STATE_LID_FRONT"
	OPEN_STATE_LID_REAR           = "OPEN_STATE_LID_REAR"
	OPEN_STATE_SERVICE_FLAP       = "OPEN_STATE_SERVICE_FLAP"
	OPEN_STATE_SPOILER            = "OPEN_STATE_SPOILER"
	OPEN_STATE_SUNROOF            = "OPEN_STATE_SUNROOF"
	OPEN_STATE_TOP                = "OPEN_STATE_TOP"
	OPEN_STATE_WINDOW_FRONT_LEFT  = "OPEN_STATE_WINDOW_FRONT_LEFT"
	OPEN_STATE_WINDOW_FRONT_RIGHT = "OPEN_STATE_WINDOW_FRONT_RIGHT"
	OPEN_STATE_WINDOW_REAR_LEFT   = "OPEN_STATE_WINDOW_REAR_LEFT"
	OPEN_STATE_WINDOW_REAR_RIGHT  = "OPEN_STATE_WINDOW_REAR_RIGHT"
	PARKING_LIGHT                 = "PARKING_LIGHT"
	RANGE                         = "RANGE"
	REMOTE_ACCESS_AUTHORIZATION   = "REMOTE_ACCESS_AUTHORIZATION"
	SERVICE_PREDICTIONS           = "SERVICE_PREDICTIONS"
	SPEED_ALARMS                  = "SPEED_ALARMS"
	SPEED_ALARMS_HISTORY          = "SPEED_ALARMS_HISTORY"
	THEFT_MODE                    = "THEFT_MODE"
	TIMERS                        = "TIMERS"
	TRIP_STATISTICS_CYCLIC        = "TRIP_STATISTICS_CYCLIC"
	TRIP_STATISTICS_LONG_TERM     = "TRIP_STATISTICS_LONG_TERM"
	TRIP_STATISTICS_SHORT_TERM    = "TRIP_STATISTICS_SHORT_TERM"
	VALET_ALARM                   = "VALET_ALARM"
	VALET_ALARM_HISTORY           = "VALET_ALARM_HISTORY"
	VTS_MODES                     = "VTS_MODES"
)

// API is an api.Vehicle implementation for Porsche PHEV cars
type MobileAPI struct {
	*request.Helper
}

// NewAPI creates a new vehicle
func NewMobileAPI(log log.Logger, identity oauth2.TokenSource) *MobileAPI {
	v := &MobileAPI{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &transport.Decorator{
		Base: &oauth2.Transport{
			Source: identity,
			Base:   v.Client.Transport,
		},
		Decorator: transport.DecorateHeaders(map[string]string{
			"apikey":      OAuth2Config.ClientID,
			"x-client-id": "52064df8-6daa-46f7-bc9e-e3232622ab26",
		}),
	}

	return v
}

// Vehicles implements the vehicle list response
func (v *MobileAPI) Vehicles() ([]StatusResponseMobile, error) {
	var res []StatusResponseMobile
	uri := fmt.Sprintf("%s/app/connect/v1/vehicles", MobileApiURI)
	err := v.GetJSON(uri, &res)
	return res, err
}

// Status implements the vehicle status response
func (v *MobileAPI) Status(vin string, items []string) (StatusResponseMobile, error) {
	var res StatusResponseMobile

	mf := make([]string, 0, len(items))
	for _, i := range items {
		mf = append(mf, "mf="+i)
	}

	uri := fmt.Sprintf("%s/app/connect/v1/vehicles/%s?%s", MobileApiURI, vin, strings.Join(mf, "&"))
	err := v.GetJSON(uri, &res)
	return res, err
}
