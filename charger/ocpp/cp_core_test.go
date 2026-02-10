package ocpp

import (
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

	// wait for timeout
	time.Sleep(100 * time.Millisecond)

	// should be connected after timeout (fallback)
	assert.True(t, cp.Connected(), "should be connected after boot timeout")
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

	// wait past the original timeout
	time.Sleep(250 * time.Millisecond)

	// should still be disconnected (timer was cancelled)
	assert.False(t, cp.Connected(), "should not be connected after cancelled timer")
}

func TestBootNotificationChannelFull(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	// fill the channel (buffer size 1)
	cp.bootNotificationRequestC <- &core.BootNotificationRequest{
		ChargePointModel: "First",
	}

	// second notification should be dropped (channel full, non-blocking send)
	bootReq := &core.BootNotificationRequest{
		ChargePointModel:  "Second",
		ChargePointVendor: "TestVendor",
	}

	res, err := cp.OnBootNotification(bootReq)
	require.NoError(t, err)
	assert.Equal(t, core.RegistrationStatusAccepted, res.Status)

	// result should still be updated even though channel was full
	assert.Equal(t, bootReq, cp.BootNotificationResult)
	assert.True(t, cp.Connected())

	// channel should have the first message (second was dropped)
	req := <-cp.bootNotificationRequestC
	assert.Equal(t, "First", req.ChargePointModel)
}

func TestReconnectAfterReboot(t *testing.T) {
	log := util.NewLogger("test")
	cp := NewChargePoint(log, "test-cp")

	// simulate initial connection + setup
	cp.connect(true)
	cp.mu.Lock()
	cp.initialized = true
	cp.mu.Unlock()

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
