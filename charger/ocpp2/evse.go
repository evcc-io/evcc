package ocpp2

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/availability"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/meter"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/remotecontrol"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/transactions"
	"github.com/lorenzodonini/ocpp-go/ocpp2.0.1/types"
)

type EVSE struct {
	mu    sync.Mutex
	log   *util.Logger
	clock clock.Clock

	st *Station
	id int

	status  *availability.StatusNotificationRequest
	statusC chan struct{}

	txnId   string
	idToken string

	meterUpdated  time.Time
	meterInterval time.Duration
	measurements  map[types.Measurand]types.SampledValue

	remoteIdToken string
}

func NewEVSE(ctx context.Context, log *util.Logger, id int, st *Station, idToken string, meterInterval time.Duration) (*EVSE, error) {
	evse := &EVSE{
		log:           log,
		st:            st,
		id:            id,
		clock:         clock.New(),
		statusC:       make(chan struct{}),
		measurements:  make(map[types.Measurand]types.SampledValue),
		remoteIdToken: idToken,
		meterInterval: meterInterval,
	}

	if err := st.registerEVSE(id, evse); err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		st.deregisterEVSE(evse.id)
	}()

	var hasStatus bool
	st.cs.WithEVSEStatus(st.ID(), id, func(s *availability.StatusNotificationRequest) {
		if _, err := evse.OnStatusNotification(s); err == nil {
			hasStatus = true
		}
	})

	if !hasStatus {
		if err := st.TriggerMessageRequest(id, remotecontrol.MessageTriggerStatusNotification); err != nil {
			log.WARN.Printf("failed triggering StatusNotification for evse %d: %v", id, err)
		}
	}

	return evse, nil
}

func (evse *EVSE) ID() int { return evse.id }

func (evse *EVSE) IdToken() string {
	evse.mu.Lock()
	defer evse.mu.Unlock()
	return evse.idToken
}

func (evse *EVSE) TransactionID() (string, error) {
	if !evse.st.Connected() {
		return "", api.ErrTimeout
	}
	evse.mu.Lock()
	defer evse.mu.Unlock()
	return evse.txnId, nil
}

func (evse *EVSE) Status() (availability.ConnectorStatus, error) {
	if !evse.st.Connected() {
		return "", api.ErrTimeout
	}
	evse.mu.Lock()
	defer evse.mu.Unlock()
	if evse.status == nil {
		return availability.ConnectorStatusUnavailable, nil
	}
	return evse.status.ConnectorStatus, nil
}

func (evse *EVSE) Initialized() error {
	trigger := time.After(Timeout / 2)
	timeout := time.After(Timeout)
	for {
		select {
		case <-evse.statusC:
			return nil
		case <-trigger:
			_ = evse.st.TriggerMessageRequest(evse.id, remotecontrol.MessageTriggerStatusNotification)
		case <-timeout:
			return api.ErrTimeout
		}
	}
}

func (evse *EVSE) NeedsAuthentication() bool {
	if !evse.st.Connected() {
		return false
	}
	evse.mu.Lock()
	defer evse.mu.Unlock()
	return evse.isWaitingForAuth()
}

func (evse *EVSE) isWaitingForAuth() bool {
	return evse.status != nil &&
		evse.txnId == "" &&
		evse.status.ConnectorStatus == availability.ConnectorStatusOccupied
}

func (evse *EVSE) isMeterTimeout() bool {
	return evse.clock.Since(evse.meterUpdated) > max(evse.meterInterval+10*time.Second, Timeout)
}

// ── Messwerte ─────────────────────────────────────────────────────────────────

func (evse *EVSE) CurrentPower() (float64, error) {
	if !evse.st.Connected() {
		return 0, api.ErrTimeout
	}
	evse.mu.Lock()
	defer evse.mu.Unlock()

	if evse.isMeterTimeout() {
		if evse.txnId != "" {
			return 0, api.ErrTimeout
		}
		return 0, nil
	}
	if m, ok := evse.measurements[types.MeasurandPowerActiveImport]; ok {
		return scaleOCPP2(m.Value, m.UnitOfMeasure), nil
	}
	if res, found := evse.phaseMeasurements(types.MeasurandPowerActiveImport); found {
		return res[0] + res[1] + res[2], nil
	}
	if evse.txnId == "" {
		return 0, nil
	}
	return 0, api.ErrNotAvailable
}

func (evse *EVSE) TotalEnergy() (float64, error) {
	if !evse.st.Connected() {
		return 0, api.ErrTimeout
	}
	evse.mu.Lock()
	defer evse.mu.Unlock()

	if evse.txnId != "" && evse.isMeterTimeout() {
		return 0, api.ErrTimeout
	}
	if m, ok := evse.measurements[types.MeasurandEnergyActiveImportRegister]; ok {
		return scaleOCPP2(m.Value, m.UnitOfMeasure) / 1e3, nil
	}
	return 0, api.ErrNotAvailable
}

func (evse *EVSE) Currents() (float64, float64, float64, error) {
	if !evse.st.Connected() {
		return 0, 0, 0, api.ErrTimeout
	}
	evse.mu.Lock()
	defer evse.mu.Unlock()

	if evse.isMeterTimeout() {
		if evse.txnId != "" {
			return 0, 0, 0, api.ErrTimeout
		}
		return 0, 0, 0, nil
	}
	if res, found := evse.phaseMeasurements(types.MeasurandCurrentImport); found {
		return res[0], res[1], res[2], nil
	}
	return 0, 0, 0, api.ErrNotAvailable
}

func (evse *EVSE) GetMaxCurrent() (float64, error) {
	if !evse.st.Connected() {
		return 0, api.ErrTimeout
	}
	evse.mu.Lock()
	defer evse.mu.Unlock()

	if evse.isMeterTimeout() {
		return 0, api.ErrTimeout
	}
	if m, ok := evse.measurements[types.MeasurandCurrentOffered]; ok {
		return scaleOCPP2(m.Value, m.UnitOfMeasure), nil
	}
	return 0, api.ErrNotAvailable
}

func (evse *EVSE) phaseMeasurements(measurand types.Measurand) ([3]float64, bool) {
	var (
		res   [3]float64
		found bool
	)
	for i := range res {
		key := types.Measurand(fmt.Sprintf("%s.L%d", measurand, i+1))
		m, ok := evse.measurements[key]
		if !ok {
			continue
		}
		found = true
		res[i] = scaleOCPP2(m.Value, m.UnitOfMeasure)
	}
	return res, found
}

func scaleOCPP2(f float64, unit *types.UnitOfMeasure) float64 {
	if unit == nil {
		return f
	}
	if unit.Multiplier != nil && *unit.Multiplier != 0 {
		exp := *unit.Multiplier
		result := f
		if exp > 0 {
			for i := 0; i < exp; i++ {
				result *= 10
			}
		} else {
			for i := 0; i > exp; i-- {
				result /= 10
			}
		}
		return result
	}
	switch {
	case strings.HasPrefix(unit.Unit, "k"):
		return f * 1e3
	case strings.HasPrefix(unit.Unit, "m"):
		return f / 1e3
	default:
		return f
	}
}

func (evse *EVSE) assumeMeterStopped() {
	evse.meterUpdated = evse.clock.Now()
	for _, measurand := range []types.Measurand{
		types.MeasurandPowerActiveImport,
		types.MeasurandCurrentImport,
	} {
		for phase := 1; phase <= 3; phase++ {
			key := types.Measurand(fmt.Sprintf("%s.L%d", measurand, phase))
			if _, ok := evse.measurements[key]; ok {
				evse.measurements[key] = types.SampledValue{Value: 0}
			}
		}
		if _, ok := evse.measurements[measurand]; ok {
			evse.measurements[measurand] = types.SampledValue{Value: 0}
		}
	}
}

// ── Message Handler ───────────────────────────────────────────────────────────

func (evse *EVSE) OnStatusNotification(req *availability.StatusNotificationRequest) (*availability.StatusNotificationResponse, error) {
	evse.mu.Lock()
	defer evse.mu.Unlock()

	if evse.status == nil {
		evse.status = req
		close(evse.statusC)
	} else {
		evse.status = req
	}

	if req.ConnectorStatus == availability.ConnectorStatusAvailable && evse.txnId != "" {
		evse.log.DEBUG.Printf("evse %d: clearing stale transaction %s on Available", evse.id, evse.txnId)
		evse.txnId = ""
		evse.idToken = ""
		evse.assumeMeterStopped()
	}

	if evse.isWaitingForAuth() && evse.remoteIdToken != "" {
		go func(token string) {
			idToken := types.IdToken{
				IdToken: token,
				Type:    types.IdTokenTypeCentral,
			}
			if err := evse.st.RequestStartTransactionRequest(evse.id, idToken); err != nil {
				evse.log.ERROR.Printf("evse %d: RequestStartTransaction: %v", evse.id, err)
			}
		}(evse.remoteIdToken)
	}

	return new(availability.StatusNotificationResponse), nil
}

func (evse *EVSE) OnTransactionEvent(req *transactions.TransactionEventRequest) (*transactions.TransactionEventResponse, error) {
	evse.mu.Lock()
	defer evse.mu.Unlock()

	for _, mv := range req.MeterValue {
		evse.applyMeterValue(mv)
	}

	switch req.EventType {
	case transactions.TransactionEventStarted:
		evse.txnId = req.TransactionInfo.TransactionID
		if req.IDToken != nil {
			evse.idToken = req.IDToken.IdToken
		}
		evse.log.DEBUG.Printf("evse %d: transaction started: %s", evse.id, evse.txnId)
	case transactions.TransactionEventUpdated:
		// Messwerte bereits oben verarbeitet
	case transactions.TransactionEventEnded:
		evse.log.DEBUG.Printf("evse %d: transaction ended: %s", evse.id, evse.txnId)
		evse.txnId = ""
		evse.idToken = ""
		evse.assumeMeterStopped()
	}

	return &transactions.TransactionEventResponse{
		IDTokenInfo: &types.IdTokenInfo{
			Status: types.AuthorizationStatusAccepted,
		},
	}, nil
}

func (evse *EVSE) OnMeterValues(req *meter.MeterValuesRequest) (*meter.MeterValuesResponse, error) {
	evse.mu.Lock()
	defer evse.mu.Unlock()
	for _, mv := range req.MeterValue {
		evse.applyMeterValue(mv)
	}
	return new(meter.MeterValuesResponse), nil
}

func (evse *EVSE) applyMeterValue(mv types.MeterValue) {
	ts := mv.Timestamp.Time
	if ts.IsZero() {
		ts = evse.clock.Now()
	}
	if ts.Before(evse.meterUpdated) {
		return
	}
	evse.meterUpdated = ts
	for _, sv := range mv.SampledValue {
		key := sv.Measurand
		if sv.Phase != "" {
			key = types.Measurand(fmt.Sprintf("%s.%s", sv.Measurand, sv.Phase))
		}
		evse.measurements[key] = sv
	}
}

func (evse *EVSE) WatchDog(ctx context.Context, timeout time.Duration) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		evse.mu.Lock()
		stale := evse.clock.Since(evse.meterUpdated) > timeout
		evse.mu.Unlock()

		if stale {
			_ = evse.st.TriggerMessageRequest(evse.id, remotecontrol.MessageTriggerMeterValues)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}