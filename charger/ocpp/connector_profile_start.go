package ocpp

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/smartcharging"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
)

const (
	profileStartSettle  = 7 * time.Second
	profileStartPoll    = 2 * time.Second
	profileStartTimeout = 20 * time.Second
)

// profileRemoteStart tracks the last acknowledged watt profile for an opted-in
// connector. Its state is protected by Connector.mu.
type profileRemoteStart struct {
	requests sync.Mutex // serializes profile writes with RemoteStart dispatch
	ctx      context.Context
	cancel   context.CancelFunc

	armed    bool
	paused   bool // preserve an acknowledged 0 W pause if MeterValues recovers the old transaction
	limit    float64
	accepted time.Time
	sequence uint64
}

func (p *profileRemoteStart) clear() {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.limit = 0
	p.sequence++
}

func (p *profileRemoteStart) reset() {
	p.clear()
	p.armed = true
	p.paused = false
}

func (conn *Connector) setStartProfile(profile *types.ChargingProfile) error {
	p := conn.profileStart
	p.requests.Lock()
	defer p.requests.Unlock()

	conn.mu.Lock()
	p.clear()
	sequence := p.sequence
	conn.mu.Unlock()

	if err := conn.cp.SetChargingProfileRequest(conn.id, profile); err != nil {
		return err
	}

	schedule := profile.ChargingSchedule
	if schedule == nil || schedule.ChargingRateUnit != types.ChargingRateUnitWatts ||
		len(schedule.ChargingSchedulePeriod) != 1 || schedule.ChargingSchedulePeriod[0].StartPeriod != 0 {
		return nil
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	if p.sequence == sequence {
		p.limit = schedule.ChargingSchedulePeriod[0].Limit
		p.accepted = time.Now()
		if p.limit == 0 {
			p.armed = true
			p.paused = true
		}
		conn.maybeStartAfterProfile()
	}
	return nil
}

// PendingStartProfile reports whether the configured remote start has a positive
// acknowledged profile, even if the charger reports zero offered current before authorization.
func (conn *Connector) PendingStartProfile() (bool, bool) {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	p := conn.profileStart
	if p == nil {
		return false, false
	}
	enabled := p.limit > 0 || (conn.txnId != 0 && !p.paused)
	return enabled, enabled || p.armed
}

// maybeStartAfterProfile must be called with Connector.mu held. It can run
// after a profile acknowledgement or after the charger enters Preparing.
func (conn *Connector) maybeStartAfterProfile() {
	p := conn.profileStart
	if p == nil || p.cancel != nil || !p.armed || !(p.limit > 0) ||
		conn.remoteIdTag == "" || !conn.isWaitingForAuth() || conn.status.ErrorCode != core.NoError {
		return
	}
	ctx, cancel := context.WithTimeout(p.ctx, profileStartTimeout)
	p.cancel = cancel
	go conn.waitForStartProfile(ctx, cancel, p.limit, p.accepted, p.sequence)
}

func (conn *Connector) waitForStartProfile(ctx context.Context, cancel context.CancelFunc, limit float64, accepted time.Time, sequence uint64) {
	defer cancel()
	timer := time.NewTimer(time.Until(accepted.Add(profileStartSettle)))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				conn.log.WARN.Printf("OCPP charging profile not confirmed: %.0f W", limit)
			}
			return
		case <-timer.C:
		}
		if ctx.Err() != nil {
			continue
		}
		if !conn.cp.Connected() {
			return
		}
		if confirmed, ok := conn.confirmStartProfile(ctx, limit); ok && conn.dispatchProfileStart(ctx, sequence, limit, confirmed) {
			return
		}
		timer.Reset(profileStartPoll)
	}
}

func (conn *Connector) confirmStartProfile(ctx context.Context, limit float64) (float64, bool) {
	requested := time.Now()
	result := make(chan *smartcharging.GetCompositeScheduleConfirmation, 1)
	if err := conn.cp.cs.GetCompositeSchedule(conn.cp.id, func(reply *smartcharging.GetCompositeScheduleConfirmation, err error) {
		if err != nil {
			reply = nil
		}
		result <- reply
	}, conn.id, 60); err != nil {
		return 0, false
	}
	select {
	case <-ctx.Done():
		return 0, false
	case reply := <-result:
		if matchesStartSchedule(reply, conn.id, limit, requested, time.Now()) {
			return reply.ChargingSchedule.ChargingSchedulePeriod[0].Limit, true
		}
		return 0, false
	}
}

func (conn *Connector) dispatchProfileStart(ctx context.Context, sequence uint64, limit, confirmed float64) bool {
	p := conn.profileStart
	p.requests.Lock()
	defer p.requests.Unlock()

	conn.mu.Lock()
	proceed := ctx.Err() == nil && p.sequence == sequence && p.armed && conn.isWaitingForAuth() &&
		conn.status.ErrorCode == core.NoError
	if proceed {
		p.armed = false
		p.paused = false
	}
	conn.mu.Unlock()
	if !proceed || !conn.cp.Connected() {
		return false
	}

	conn.log.DEBUG.Printf("OCPP charging profile confirmed: %.0f W (requested %.0f W)", confirmed, limit)
	if err := conn.RemoteStartTransactionRequest(conn.remoteIdTag); err != nil {
		conn.log.WARN.Printf("RemoteStartTransaction after charging profile: %v", err)
	}
	return true
}

func matchesStartSchedule(reply *smartcharging.GetCompositeScheduleConfirmation, connector int, limit float64, requested, now time.Time) bool {
	if reply == nil || reply.Status != smartcharging.GetCompositeScheduleStatusAccepted ||
		reply.ConnectorId == nil || *reply.ConnectorId != connector || reply.ScheduleStart == nil ||
		reply.ScheduleStart.Before(requested.Add(-profileStartPoll)) || reply.ScheduleStart.After(now.Add(profileStartPoll)) {
		return false
	}

	schedule := reply.ChargingSchedule
	if schedule == nil || schedule.ChargingRateUnit != types.ChargingRateUnitWatts || len(schedule.ChargingSchedulePeriod) == 0 {
		return false
	}
	period := schedule.ChargingSchedulePeriod[0]
	if period.StartPeriod != 0 || !(period.Limit > 0) {
		return false
	}

	// The SCharger-22KT-S0 reports its nominal 11 kW for a 16 A three-phase profile.
	return math.Abs(period.Limit-limit) < 0.1 || (limit == 11040 && period.Limit == 11000)
}
