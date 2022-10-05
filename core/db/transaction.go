package db

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/fatih/structs"
)

// Transaction is a single charging transaction
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

// Transactions is a list of transactions
type Transactions []Transaction

var _ api.CsvWriter = (*Transactions)(nil)

func (t *Transactions) writeHeader(ww *csv.Writer) {
	var row []string
	for _, f := range structs.Fields(Transaction{}) {
		if f.Tag("json") == "-" {
			continue
		}
		row = append(row, f.Name())
	}
	_ = ww.Write(row)
}

func (t *Transactions) writeRow(ww *csv.Writer, r Transaction) {
	var row []string
	for _, f := range structs.Fields(r) {
		if f.Tag("json") == "-" {
			continue
		}

		val := fmt.Sprintf("%v", f.Value())

		switch v := f.Value().(type) {
		case float64:
			val = strconv.FormatFloat(v, 'f', 3, 64)
		case time.Time:
			if v.IsZero() {
				val = ""
			} else {
				val = v.Local().Format("2006-01-02 15:04:05")
			}
		}

		row = append(row, val)
	}
	_ = ww.Write(row)
}

// WriteCsv implements the api.CsvWriter interface
func (t *Transactions) WriteCsv(w io.Writer) {
	ww := csv.NewWriter(w)
	t.writeHeader(ww)

	for _, r := range *t {
		t.writeRow(ww, r)
	}

	ww.Flush()
}
