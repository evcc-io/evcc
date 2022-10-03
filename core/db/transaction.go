package db

import (
	"time"
)

// Transaction is a single charging transaction with status and reservation and payment data
type Transaction struct {
	ID            uint      `json:"-" gorm:"primarykey"`
	Created       time.Time `json:"created"`
	Finished      time.Time `json:"finished"`
	Loadpoint     string    `json:"loadpoint"`
	Identifier    string    `json:"identifier"`
	Vehicle       string    `json:"vehicle"`
	MeterStart    float64   `json:"meterStart" gorm:"column:meter_start_kwh"`
	MeterStop     float64   `json:"meterStop" gorm:"column:meter_end_kwh"`
	ChargedEnergy float64   `json:"chargedEnergy" gorm:"column:charged_kwh"`
}

// Stop stops charging session with end meter reading and due total amount
func (t *Transaction) Stop(chargedWh, total float64) {
	t.ChargedEnergy = chargedWh / 1e3
	t.MeterStop = total
	t.Finished = time.Now()
}
