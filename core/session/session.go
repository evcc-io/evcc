package session

import (
	"context"
	"io"
	"time"

	"github.com/evcc-io/evcc/api"
	csvutil "github.com/evcc-io/evcc/util/csv"
)

// Session is a single charging session
type Session struct {
	ID              uint           `json:"id" csv:"-" gorm:"primarykey"`
	Created         time.Time      `json:"created"`
	Finished        time.Time      `json:"finished"`
	Loadpoint       string         `json:"loadpoint"`
	Identifier      string         `json:"identifier"`
	Vehicle         string         `json:"vehicle"`
	Odometer        *float64       `json:"odometer" format:"int"`
	MeterStart      *float64       `json:"meterStart" csv:"Meter Start (kWh)" gorm:"column:meter_start_kwh"`
	MeterStop       *float64       `json:"meterStop" csv:"Meter Stop (kWh)" gorm:"column:meter_end_kwh"`
	ChargedEnergy   float64        `json:"chargedEnergy" csv:"Charged Energy (kWh)" gorm:"column:charged_kwh"`
	ChargeDuration  *time.Duration `json:"chargeDuration" csv:"Charge Duration" gorm:"column:charge_duration"`
	SolarPercentage *float64       `json:"solarPercentage" csv:"Solar (%)" gorm:"column:solar_percentage"`
	Price           *float64       `json:"price" csv:"Price" gorm:"column:price"`
	PricePerKWh     *float64       `json:"pricePerKWh" csv:"Price/kWh" gorm:"column:price_per_kwh"`
	Co2PerKWh       *float64       `json:"co2PerKWh" csv:"CO2/kWh (gCO2eq)" gorm:"column:co2_per_kwh"`
}

// Sessions is a list of sessions
type Sessions []Session

var _ api.CsvWriter = (*Sessions)(nil)

// WriteCsv implements the api.CsvWriter interface
func (t *Sessions) WriteCsv(ctx context.Context, w io.Writer) error {
	return csvutil.WriteStructSlice(ctx, w, t, csvutil.Config{
		I18nPrefix: "sessions.csv",
	})
}
