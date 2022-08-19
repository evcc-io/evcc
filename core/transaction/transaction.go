package transaction

import (
	"time"
)

// Transaction is a single charging transaction with status and reservation and payment data
type Transaction struct {
	ID          uint `gorm:"primarykey"`
	LoadpointId int
	Loadpoint   string
	VehicleId   int
	Vehicle     string
	StartTime   time.Time
	EndTime     time.Time
	StartkWh    float64 `gorm:"column:start_kwh"`
	EndkWh      float64 `gorm:"column:end_kwh"`
	ChargedkWh  float64 `gorm:"column:charged_kwh"`
}

// New creates a charging transaction given email and payment reservation
func New(id int, email, rfid string, pricePerkWh, reservationAmount int, receipt, trace []byte) *Transaction {
	t := Transaction{}

	return &t
}

// Start starts charging session with start meter reading
func (t *Transaction) Start(total float64) {
	t.StartkWh = total
	t.StartTime = time.Now()
}

// Stop stops charging session with end meter reading and due total amount
func (t *Transaction) Stop(charged, total float64) {
	t.ChargedkWh = charged
	t.EndkWh = total
	t.EndTime = time.Now()
}
