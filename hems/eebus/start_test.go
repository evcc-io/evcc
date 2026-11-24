package eebus

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/enbility/eebus-go/usecases/mocks"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// fakeService is an EEBus server without any Energy Guard connecting
type fakeService struct {
	mu                     sync.Mutex
	cs                     *eebus.ControllableSystem
	registered, unregister []string
}

func (s *fakeService) RegisterDevice(ski, _ string, _ eebus.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registered = append(s.registered, ski)
	return nil
}

func (s *fakeService) UnregisterDevice(ski string, _ eebus.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unregister = append(s.unregister, ski)
}

func (s *fakeService) unregistered() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.unregister)
}

func (s *fakeService) ControllableSystem() *eebus.ControllableSystem {
	return s.cs
}

// withService replaces the EEBus server for the duration of the test
func withService(t *testing.T) *fakeService {
	t.Helper()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	lpc := mocks.NewCsLPCInterface(t)
	lpp := mocks.NewCsLPPInterface(t)
	lpc.EXPECT().RemoteEntitiesScenarios().Return(nil).Maybe()
	lpp.EXPECT().RemoteEntitiesScenarios().Return(nil).Maybe()
	lpc.EXPECT().SetFailsafeConsumptionActivePowerLimit(testFailsafeConsumption, true).Return(nil).Once()
	lpc.EXPECT().SetFailsafeDurationMinimum(testFailsafeDuration, true).Return(nil).Once()
	lpc.EXPECT().SetConsumptionLimit(mock.Anything).Return(nil).Maybe()
	lpp.EXPECT().SetProductionNominalMax(testProductionNominal).Return(nil).Once()
	lpp.EXPECT().SetFailsafeProductionActivePowerLimit(testFailsafeProduction, true).Return(nil).Once()
	lpp.EXPECT().SetFailsafeDurationMinimum(testFailsafeDuration, true).Return(nil).Once()
	lpp.EXPECT().SetProductionLimit(mock.Anything).Return(nil).Maybe()

	svc := &fakeService{cs: &eebus.ControllableSystem{CsLPCInterface: lpc, CsLPPInterface: lpp}}

	prev := instance
	instance = func() (service, error) { return svc, nil }
	t.Cleanup(func() { instance = prev })

	return svc
}

func testLimits() Limits {
	failsafeProduction := testFailsafeProduction
	return Limits{
		FailsafeConsumptionActivePowerLimit: testFailsafeConsumption,
		ProductionNominalMax:                testProductionNominal,
		FailsafeProductionActivePowerLimit:  &failsafeProduction,
		FailsafeDurationMinimum:             testFailsafeDuration,
	}
}

// TestNewEEBus_StartsWithoutEnergyGuard covers [LPC-901/1]: startup neither waits for
// nor fails without the Energy Guard, and "init" applies the failsafe limits.
func TestNewEEBus_StartsWithoutEnergyGuard(t *testing.T) {
	svc := withService(t)

	start := time.Now()
	c, err := NewEEBus(t.Context(), "abcd", testLimits(), nil, &stubSite{}, time.Millisecond)
	require.NoError(t, err)
	assert.Less(t, time.Since(start), time.Second, "startup must not wait for the Energy Guard")
	assert.Equal(t, []string{"abcd"}, svc.registered, "the Energy Guard stays trusted")
	assert.Empty(t, svc.unregistered())

	require.NoError(t, c.run())
	assert.Equal(t, StatusFailsafe, c.status)
	assertConsumptionLimit(t, c, testFailsafeConsumption)

	production := c.MaxProductionPower()
	require.NotNil(t, production)
	assert.Equal(t, testFailsafeProduction, *production)
}

// TestNewEEBus_UnregistersWhenContextEnds: the device is unregistered when its context ends
func TestNewEEBus_UnregistersWhenContextEnds(t *testing.T) {
	svc := withService(t)

	ctx, cancel := context.WithCancel(context.Background())

	c, err := NewEEBus(ctx, "abcd", testLimits(), nil, &stubSite{}, time.Millisecond)
	require.NoError(t, err)
	assert.Empty(t, svc.unregistered())

	cancel()
	assert.Eventually(t, func() bool { return len(svc.unregistered()) == 1 }, time.Second, time.Millisecond)

	// Wait still reports the Energy Guard as unreachable
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer waitCancel()
	require.ErrorIs(t, c.Wait(waitCtx), context.DeadlineExceeded)
}
