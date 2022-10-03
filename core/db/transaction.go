package db

import (
	"time"
)

// Transaction is a single charging transaction with status and reservation and payment data
type Transaction struct {
	ID            uint `gorm:"primarykey"`
	Loadpoint     string
	Identifier    string
	Vehicle       string
	StartTime     time.Time
	EndTime       time.Time
	MeterStart    float64 `gorm:"column:meter_start_kwh"`
	MeterStop     float64 `gorm:"column:meter_end_kwh"`
	ChargedEnergy float64 `gorm:"column:charged_kwh"`
}

// Stop stops charging session with end meter reading and due total amount
func (t *Transaction) Stop(chargedWh, total float64) {
	t.ChargedEnergy = chargedWh / 1e3
	t.MeterStop = total
	t.EndTime = time.Now()
}
