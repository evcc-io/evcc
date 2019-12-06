package core

import (
	"sync"
	"sync/atomic"

	"github.com/andig/evcc/api"
)

// State provides synchronized access to guarded variables
// NOTE: State uses int32 instead of bool for mutex-free atomic bool access
// as demonstrated in https://github.com/tevino/abool/blob/master/bool.go
type State struct {
	sync.RWMutex
	mode          api.ChargeMode   // current charge mode
	status        api.ChargeStatus // current connection state
	charging      int32            // current charging state
	targetCurrent int64            // previous target current
	chargePower   float64          // previous charge power
	socCharge     float64          // previous soc charge
}

// Mode synchronizes access to mode
func (s *State) Mode() api.ChargeMode {
	s.RLock()
	defer s.RUnlock()
	return s.mode
}

// SetMode synchronizes access to mode
func (s *State) SetMode(mode api.ChargeMode) {
	s.Lock()
	defer s.Unlock()
	s.mode = mode
}

// Status synchronizes access to status
func (s *State) Status() api.ChargeStatus {
	s.Lock()
	defer s.Unlock()
	return s.status
}

// SetStatus synchronizes access to status
func (s *State) SetStatus(status api.ChargeStatus) {
	s.Lock()
	defer s.Unlock()
	s.status = status
}

// Charging synchronizes access to charging
func (s *State) Charging() bool {
	return atomic.LoadInt32(&s.charging) == 1
}

// SetCharging synchronizes access to charging
func (s *State) SetCharging(charging bool) {
	if charging {
		atomic.StoreInt32(&s.charging, 1)
	} else {
		atomic.StoreInt32(&s.charging, 0)
	}
}

// TargetCurrent synchronizes access to targetCurrent
func (s *State) TargetCurrent() int64 {
	return atomic.LoadInt64(&s.targetCurrent)
}

// SetTargetCurrent synchronizes access to targetCurrent
func (s *State) SetTargetCurrent(targetCurrent int64) {
	atomic.StoreInt64(&s.targetCurrent, targetCurrent)
}

// ChargePower synchronizes access to chargePower
func (s *State) ChargePower() float64 {
	s.RLock()
	defer s.RUnlock()
	return s.chargePower
}

// SetChargePower synchronizes access to chargePower
func (s *State) SetChargePower(chargePower float64) {
	s.Lock()
	defer s.Unlock()
	s.chargePower = chargePower
}

// SocCharge synchronizes access to socCharge
func (s *State) SocCharge() float64 {
	s.RLock()
	defer s.RUnlock()
	return s.socCharge
}

// SetSocCharge synchronizes access to socCharge
func (s *State) SetSocCharge(socCharge float64) {
	s.Lock()
	defer s.Unlock()
	s.socCharge = socCharge
}
