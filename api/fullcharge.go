package api

import "errors"

// FullCharge configures a periodic charge to 100%, e.g. for LFP cell balancing.
// A full charge is due once either configured threshold is reached.
type FullCharge struct {
	Days   int     `json:"days"`   // days since the last full charge, 0 disables
	Energy float64 `json:"energy"` // kWh charged since the last full charge, 0 disables
}

// Enabled returns true if any threshold is configured
func (fc FullCharge) Enabled() bool {
	return fc.Days > 0 || fc.Energy > 0
}

// Validate checks the thresholds
func (fc FullCharge) Validate() error {
	if fc.Days < 0 || fc.Days > 365 {
		return errors.New("days out of range")
	}
	if fc.Energy < 0 || fc.Energy > 10000 {
		return errors.New("energy out of range")
	}
	return nil
}
