package ocpp

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootNotificationStoresResultAndConnects(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	assert.False(t, cp.Connected(), "should not be connected initially")
	assert.Nil(t, cp.BootNotificationResult, "should have no boot result initially")

	bootReq := &core.BootNotificationRequest{
		ChargePointModel:  "TestModel",
		ChargePointVendor: "TestVendor",
	}

	res, err := cp.OnBootNotification(bootReq)
	require.NoError(t, err)
	assert.Equal(t, core.RegistrationStatusAccepted, res.Status)

	// should be connected after BootNotification
	assert.True(t, cp.Connected(), "should be connected after BootNotification")
	assert.Equal(t, bootReq, cp.BootNotificationResult, "should store boot result")

	// should have sent to channel
	select {
	case req := <-cp.bootNotificationRequestC:
		assert.Equal(t, bootReq, req)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("bootNotificationRequestC should have received notification")
	}
}

func TestBootNotificationStopsTimer(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	// simulate WebSocket connect (starts timer)
	cp.onTransportConnect()

	// timer should be set
	cp.mu.RLock()
	assert.NotNil(t, cp.bootTimer, "boot timer should be active after transport connect")
	cp.mu.RUnlock()

	// BootNotification arrives - should stop timer and connect
	bootReq := &core.BootNotificationRequest{
		ChargePointModel:  "TestModel",
		ChargePointVendor: "TestVendor",
	}
	_, err := cp.OnBootNotification(bootReq)
	require.NoError(t, err)

	assert.True(t, cp.Connected(), "should be connected after BootNotification")

	cp.mu.RLock()
	assert.Nil(t, cp.bootTimer, "boot timer should be nil after BootNotification")
	cp.mu.RUnlock()

	// drain the channel
	<-cp.bootNotificationRequestC
}

func TestTransportConnectTimeoutFallback(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	// use a short timeout for testing
	origTimeout := Timeout
	Timeout = 50 * time.Millisecond
	defer func() { Timeout = origTimeout }()

	assert.False(t, cp.Connected(), "should not be connected initially")

	cp.onTransportConnect()

	// should not be connected yet
	assert.False(t, cp.Connected(), "should not be connected immediately after transport connect")

	// should be connected after timeout (fallback)
	require.Eventually(t, cp.Connected, time.Second, 10*time.Millisecond,
		"should be connected after boot timeout")

	// boot timer should be cleared after firing
	cp.mu.RLock()
	assert.Nil(t, cp.bootTimer, "boot timer should be nil after timeout")
	cp.mu.RUnlock()
}

func TestDisconnectCancelsTimer(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	// use a short timeout for testing
	origTimeout := Timeout
	Timeout = 200 * time.Millisecond
	defer func() { Timeout = origTimeout }()

	cp.onTransportConnect()

	cp.mu.RLock()
	assert.NotNil(t, cp.bootTimer, "boot timer should be active")
	cp.mu.RUnlock()

	// disconnect should cancel timer
	cp.connect(false)

	cp.mu.RLock()
	assert.Nil(t, cp.bootTimer, "boot timer should be nil after disconnect")
	cp.mu.RUnlock()

	// should still be disconnected (timer was cancelled)
	require.Never(t, cp.Connected, 300*time.Millisecond, 10*time.Millisecond,
		"should not be connected after cancelled timer")
}

func TestBootNotificationChannelCoalesces(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	// pre-fill the channel (buffer size 1) with a stale notification
	cp.bootNotificationRequestC <- &core.BootNotificationRequest{
		ChargePointModel: "Stale",
	}

	// a fresh BootNotification must replace the stale one rather than be
	// discarded - the consumer needs the charge point's current state
	bootReq := &core.BootNotificationRequest{
		ChargePointModel:  "Fresh",
		ChargePointVendor: "TestVendor",
	}

	res, err := cp.OnBootNotification(bootReq)
	require.NoError(t, err)
	assert.Equal(t, core.RegistrationStatusAccepted, res.Status)

	assert.Equal(t, bootReq, cp.BootNotificationResult)
	assert.True(t, cp.Connected())

	// channel must hold the freshest notification, exactly once
	req := <-cp.bootNotificationRequestC
	assert.Equal(t, "Fresh", req.ChargePointModel)

	select {
	case extra := <-cp.bootNotificationRequestC:
		t.Fatalf("channel should hold exactly one notification, got extra: %s", extra.ChargePointModel)
	default:
	}
}

// TestBootNotificationRebootLoopKeepsLatest reproduces the EN+ reboot loop from
// issue #30113: a charge point reconnects repeatedly, sending a BootNotification
// each time, before the reboot monitor consumes any of them. The buffered
// channel must never overflow and must always retain the most recent
// notification so a later Setup re-runs against the charge point's real state.
func TestBootNotificationRebootLoopKeepsLatest(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	for i := range 5 {
		_, err := cp.OnBootNotification(&core.BootNotificationRequest{
			ChargePointModel: "EN+",
			FirmwareVersion:  fmt.Sprintf("1.1.%d", i),
		})
		require.NoError(t, err)
	}

	// exactly one notification queued, and it is the latest
	req := <-cp.bootNotificationRequestC
	assert.Equal(t, "1.1.4", req.FirmwareVersion)

	select {
	case extra := <-cp.bootNotificationRequestC:
		t.Fatalf("channel must buffer at most one notification, got extra: %s", extra.FirmwareVersion)
	default:
	}
}

func TestReconnectAfterReboot(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	// simulate initial connection
	cp.connect(true)

	// simulate disconnect
	cp.connect(false)
	assert.False(t, cp.Connected())

	// simulate reconnect with BootNotification (reboot)
	cp.onTransportConnect()

	bootReq := &core.BootNotificationRequest{
		ChargePointModel:  "TestModel",
		ChargePointVendor: "TestVendor",
	}
	_, err := cp.OnBootNotification(bootReq)
	require.NoError(t, err)

	assert.True(t, cp.Connected(), "should be connected after reboot BootNotification")

	// channel should have the notification for monitorReboot to pick up
	select {
	case req := <-cp.bootNotificationRequestC:
		assert.Equal(t, bootReq, req)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("bootNotificationRequestC should have received reboot notification")
	}
}

func TestMonitorRebootOnlyOnce(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	ctx := t.Context()
	var callCount atomic.Int32
	setup := func() error { callCount.Add(1); return nil }

	cp.MonitorReboot(ctx, setup)
	cp.MonitorReboot(ctx, setup)
	cp.MonitorReboot(ctx, setup)

	// send a boot notification to trigger the monitor
	cp.bootNotificationRequestC <- &core.BootNotificationRequest{
		ChargePointModel: "TestModel",
	}

	// wait for the goroutine to process it
	require.Eventually(t, func() bool { return callCount.Load() == 1 }, time.Second, 10*time.Millisecond,
		"setup should be called exactly once")
}
