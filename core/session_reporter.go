package core

import "github.com/evcc-io/evcc/core/session"

// SessionReporter receives the session boundaries and energy register of each
// loadpoint for outbound reporting. Implemented outside core and set at boot,
// so core does not depend on the reporting protocol.
type SessionReporter interface {
	SessionStart(loadpointId int, meterStartWh float64)
	SessionMeter(loadpointId int, registerWh float64)
	SessionStop(loadpointId int, meterStopWh float64)
}

// SetSessionReporter sets the outbound session reporter, nil disables reporting.
// Loadpoints keep their own reference so the session code never reaches the site.
func (site *Site) SetSessionReporter(r SessionReporter) {
	for _, lp := range site.loadpoints {
		if lp != nil {
			lp.sessionReporter = r
		}
	}
}

func (lp *Loadpoint) reporter() SessionReporter {
	return lp.sessionReporter
}

// sessionRegisterKWh is the absolute energy register reported for the session: the
// charger's meter when it has one, otherwise the session energy on top of the start
// reading. Samples and the stop value both come from here, so they agree.
func (lp *Loadpoint) sessionRegisterKWh(s *session.Session) float64 {
	if m := lp.chargeMeterTotal(); m > 0 {
		return m
	}

	register := s.ChargedEnergy
	if s.MeterStart != nil {
		register += *s.MeterStart
	}
	return register
}
