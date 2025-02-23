package openmeteo

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// URIs, production and sandbox
// see https://open-meteo.com/en/docs

const (
	BaseURL     = "https://api.open-meteo.com/"
	MinInterval = 2 * time.Minute    // Minimum interval of 15 minutes
	TimeFormat  = "2006-01-02T15:04" // RFC3339 short
)

///////////////////////////////////////////////////////////////////////////////////////////
///// DEBUG VERISON
///////////////////////////////////////////////////////////////////////////////////////////

type OpenMeteo struct {
	*request.Helper
	site             string
	log              *util.Logger
	Data             *util.Monitor[api.Rates]
	Latitude         []float64
	Longitude        []float64
	Azimuth          []float64
	Declination      []float64
	DcKwp            []float64
	AcKwp            float64
	ApiKey           string
	BaseURL          string
	WeatherModel     string
	DampingMorning   []float64
	DampingEvening   []float64
	EfficiencyFactor []float64
	PastDays         int
	ForecastDays     int
	Interval         time.Duration
}

type Response struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// GenerationTimeMs      float64 `json:"generationtime_ms"`
	UtcOffsetSeconds int    `json:"utc_offset_seconds"`
	Location         string `json:"timezone"`
	// TimezoneAbbreviation  string  `json:"timezone_abbreviation"`
	// Elevation             float64 `json:"elevation"`
	// Minutely15Units       struct {
	//     Time                      string `json:"time"`
	//     Temperature2m             string `json:"temperature_2m"`
	//     GlobalTiltedIrradiance    string `json:"global_tilted_irradiance"`
	//     GlobalTiltedIrradianceInst string `json:"global_tilted_irradiance_instant"`
	// } `json:hourly_units"`
	// Minutely15
	Hourly struct {
		Time                   []string  `json:"time"`
		Temperature2m          []float64 `json:"temperature_2m"`
		GlobalTiltedIrradiance []float64 `json:"global_tilted_irradiance"`
		// GlobalTiltedIrradianceInst []float64 `json:"global_tilted_irradiance_instant"`
	} `json:"hourly"`
	Daily struct {
		Sunrise []string `json:"sunrise"`
		Sunset  []string `json:"sunset"`
	} `json:"daily"`
}

// NewOpenMeteo creates a new OpenMeteo instance
func NewOpenMeteo(log *util.Logger, cc OpenMeteo) *OpenMeteo {
	// Ensure the interval is not less than 15 minutes
	if cc.Interval < MinInterval {
		cc.Interval = MinInterval
	}

	return &OpenMeteo{
		Helper:           request.NewHelper(log),
		log:              log,
		Latitude:         cc.Latitude,
		Longitude:        cc.Longitude,
		Azimuth:          cc.Azimuth,
		Declination:      cc.Declination,
		DcKwp:            cc.DcKwp,
		AcKwp:            cc.AcKwp,
		ApiKey:           cc.ApiKey,
		BaseURL:          cc.BaseURL,
		WeatherModel:     cc.WeatherModel,
		DampingMorning:   cc.DampingMorning,
		DampingEvening:   cc.DampingEvening,
		EfficiencyFactor: cc.EfficiencyFactor,
		PastDays:         cc.PastDays,
		ForecastDays:     cc.ForecastDays,
		Interval:         cc.Interval,
		Data:             util.NewMonitor[api.Rates](2 * time.Minute),
	}
}
