package ocpp

import (
	"fmt"
	"strconv"
	"strings"
	// "sync"
	"time"

	"github.com/evcc-io/evcc/api"
	// "github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/types"
	// "github.com/lorenzodonini/ocpp-go/ocpp1.6/remotetrigger"
)

// core handlers used to handle metering

func (cp *CP) MeterValues(request *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	cp.log.TRACE.Printf("%T: %+v", request, request)
	// cp.log.TRACE.Printf("Meter Values TransactionId: %d", *request.TransactionId)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.updated = time.Now()

	for _, meterValue := range request.MeterValue {
		// ignore old meter value requests
		if meterValue.Timestamp.Time.After(cp.meterUpdated) {
			for _, sample := range meterValue.SampledValue {
				cp.measurements[getSampleKey(sample)] = sample
				cp.meterUpdated = time.Now()
			}
		}
	}

	return new(core.MeterValuesConfirmation), nil
}

func getSampleKey(s types.SampledValue) string {
	if s.Phase != "" {
		return string(s.Measurand) + "@" + string(s.Phase)
	}

	return string(s.Measurand)
}

// this watchdog routine triggers meter values messages if older than timeout.
// Must be wrapped in a goroutine.
func (cp *CP) MeteringWatchdog(timeout time.Duration) {
	for ; true; <-time.NewTicker(timeout).C {
		cp.mu.Lock()
		update := cp.currentTransaction != nil && time.Since(cp.meterUpdated) > timeout
		cp.mu.Unlock()

		if update {
			Instance().TriggerMessageRequest(cp.ID(), core.MeterValuesFeatureName)
		}
	}
}

// metering APIs implementations

var _ api.Meter = (*CP)(nil)

func (cp *CP) CurrentPower() (float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.timeout > 0 && time.Since(cp.meterUpdated) > cp.timeout {
		return 0, api.ErrNotAvailable
	}

	if m, ok := cp.measurements[string(types.MeasurandPowerActiveImport)]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit), err
	}

	return 0, api.ErrNotAvailable
}

var _ api.MeterEnergy = (*CP)(nil)

func (cp *CP) TotalEnergy() (float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.timeout > 0 && time.Since(cp.meterUpdated) > cp.timeout {
		return 0, api.ErrNotAvailable
	}

	if m, ok := cp.measurements[string(types.MeasurandEnergyActiveImportRegister)]; ok {
		f, err := strconv.ParseFloat(m.Value, 64)
		return scale(f, m.Unit) / 1e3, err
	}

	return 0, api.ErrNotAvailable
}

func scale(f float64, scale types.UnitOfMeasure) float64 {
	switch {
	case strings.HasPrefix(string(scale), "k"):
		return f * 1e3
	case strings.HasPrefix(string(scale), "m"):
		return f / 1e3
	default:
		return f
	}
}

func getKeyCurrentPhase(phase int) string {
	return string(types.MeasurandCurrentImport) + "@L" + strconv.Itoa(phase)
}

var _ api.MeterCurrent = (*CP)(nil)

func (cp *CP) Currents() (float64, float64, float64, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.timeout > 0 && time.Since(cp.meterUpdated) > cp.timeout {
		return 0, 0, 0, api.ErrNotAvailable
	}

	currents := make([]float64, 0, 3)

	for phase := 1; phase <= 3; phase++ {
		m, ok := cp.measurements[getKeyCurrentPhase(phase)]
		if !ok {
			return 0, 0, 0, api.ErrNotAvailable
		}

		f, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid current for phase %d: %w", phase, err)
		}

		currents = append(currents, scale(f, m.Unit))
	}

	return currents[0], currents[1], currents[2], nil
}
