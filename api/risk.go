package api

import "fmt"

// RiskProfile is the forecast uncertainty the optimizer hedges against
type RiskProfile struct {
	Solar  float64 `json:"solar"`  // share of the solar forecast a slot may fall short of, 0..1
	Demand float64 `json:"demand"` // share of the demand forecast a slot may exceed, 0..1
	Budget float64 `json:"budget"` // 15min slots per day that deviate adversely, 0 < budget <= 96
}

// Validate checks the value ranges
func (p RiskProfile) Validate() error {
	if p.Solar < 0 || p.Solar > 1 {
		return fmt.Errorf("invalid solar share: %g", p.Solar)
	}
	if p.Demand < 0 || p.Demand > 1 {
		return fmt.Errorf("invalid demand share: %g", p.Demand)
	}
	if p.Budget <= 0 || p.Budget > 96 {
		return fmt.Errorf("invalid budget: %g", p.Budget)
	}
	return nil
}
