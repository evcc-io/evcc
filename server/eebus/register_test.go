package eebus

import (
	"testing"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	eebusmocks "github.com/enbility/eebus-go/mocks"
	spineapi "github.com/enbility/spine-go/api"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// mockDevice implements Device for testing
type mockDevice struct {
	connectCalled  bool
	connectValue   bool
	useCaseEvents  []DeviceEntity
	deviceEntities []DeviceEntity // returned by DeviceEntities
}

func (d *mockDevice) Connect(connected bool) {
	d.connectCalled = true
	d.connectValue = connected
}

func (d *mockDevice) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	d.useCaseEvents = append(d.useCaseEvents, DeviceEntity{Entity: entity, Event: event})
}

func (d *mockDevice) DeviceEntities() []DeviceEntity {
	return d.deviceEntities
}

var _ Device = (*mockDevice)(nil)
var _ StatefulDevice = (*mockDevice)(nil)

func TestRegisterDevice_AlreadyConnected(t *testing.T) {
	c := &EEBus{
		log:       util.NewLogger("test"),
		clients:   make(map[string][]Device),
		connected: make(map[string]bool),
	}

	// simulate existing connection
	c.connected["aabbcc"] = true

	newDevice := &mockDevice{}
	c.registerDevice("aabbcc", newDevice)

	assert.True(t, newDevice.connectCalled, "new device should receive Connect")
	assert.True(t, newDevice.connectValue, "new device should receive Connect(true)")
}

func TestRegisterDevice_NotConnected(t *testing.T) {
	c := &EEBus{
		log:       util.NewLogger("test"),
		clients:   make(map[string][]Device),
		connected: make(map[string]bool),
	}

	newDevice := &mockDevice{}
	c.registerDevice("aabbcc", newDevice)

	assert.False(t, newDevice.connectCalled, "new device should not receive Connect when SKI not connected")
}

func TestRegisterDevice_TransfersEntityState(t *testing.T) {
	c := &EEBus{
		log:       util.NewLogger("test"),
		clients:   make(map[string][]Device),
		connected: make(map[string]bool),
	}

	evEntity := spinemocks.NewEntityRemoteInterface(t)

	// old device with entity state
	oldDevice := &mockDevice{
		deviceEntities: []DeviceEntity{
			{Entity: evEntity, Event: "test:evConnected"},
		},
	}
	c.clients["aabbcc"] = []Device{oldDevice}
	c.connected["aabbcc"] = true

	// register new device on same SKI
	newDevice := &mockDevice{}
	c.registerDevice("aabbcc", newDevice)

	assert.True(t, newDevice.connectCalled)
	assert.Len(t, newDevice.useCaseEvents, 1, "new device should receive entity state from old device")
	assert.Equal(t, evEntity, newDevice.useCaseEvents[0].Entity)
	assert.Equal(t, eebusapi.EventType("test:evConnected"), newDevice.useCaseEvents[0].Event)
}

func TestRegisterDevice_NoTransferWhenNoEntities(t *testing.T) {
	c := &EEBus{
		log:       util.NewLogger("test"),
		clients:   make(map[string][]Device),
		connected: make(map[string]bool),
	}

	// old device without entity state (e.g. no EV connected)
	oldDevice := &mockDevice{deviceEntities: nil}
	c.clients["aabbcc"] = []Device{oldDevice}
	c.connected["aabbcc"] = true

	newDevice := &mockDevice{}
	c.registerDevice("aabbcc", newDevice)

	assert.True(t, newDevice.connectCalled)
	assert.Empty(t, newDevice.useCaseEvents, "no entity state to transfer")
}

func TestConnect_TracksState(t *testing.T) {
	c := &EEBus{
		log:       util.NewLogger("test"),
		clients:   make(map[string][]Device),
		connected: make(map[string]bool),
	}

	dev := &mockDevice{}
	c.clients["aabbcc"] = []Device{dev}

	c.connect("aabbcc", true)
	assert.True(t, c.connected["aabbcc"])

	c.connect("aabbcc", false)
	assert.False(t, c.connected["aabbcc"])
}

// TestUnregisterDevice_LastClientTearsDownShip pins the basic teardown
// contract: removing the last client for an SKI must call
// UnregisterRemoteSKI so the SHIP session is released.
func TestUnregisterDevice_LastClientTearsDownShip(t *testing.T) {
	service := eebusmocks.NewServiceInterface(t)
	service.EXPECT().UnregisterRemoteSKI("aabbcc").Once()

	dev := &mockDevice{}
	c := &EEBus{
		log:       util.NewLogger("test"),
		service:   service,
		clients:   map[string][]Device{"aabbcc": {dev}},
		connected: make(map[string]bool),
	}

	c.UnregisterDevice("aabbcc", dev)

	assert.Len(t, c.clients["aabbcc"], 0, "client list not empty")
}

// TestUnregisterDevice_KeepsShipWhenOtherClientsRemain pins that teardown
// is skipped while another client is still registered on the SKI — this
// is the update path where the old instance is retired after the new one
// has been installed via registerDevice's replay.
func TestUnregisterDevice_KeepsShipWhenOtherClientsRemain(t *testing.T) {
	service := eebusmocks.NewServiceInterface(t)
	// no UnregisterRemoteSKI expectation — any call would fail the mock

	dev1 := &mockDevice{}
	dev2 := &mockDevice{}
	c := &EEBus{
		log:       util.NewLogger("test"),
		service:   service,
		clients:   map[string][]Device{"aabbcc": {dev1, dev2}},
		connected: make(map[string]bool),
	}

	c.UnregisterDevice("aabbcc", dev1)

	assert.Equal(t, []Device{dev2}, c.clients["aabbcc"])
}

// TestUnregisterDevice_MutexNotHeldDuringShipCall is the regression guard
// for issue #28942. It asserts that c.mux is NOT held at the point
// UnregisterRemoteSKI is called. The pre-fix code held c.mux across that
// cross-layer call, and ship-go's synchronous HandleConnectionClosed
// callback chain re-entered connect(ski, false) on the same goroutine,
// which then deadlocked on c.mux.Lock() (Go mutexes are non-reentrant).
//
// The assertion uses a goroutine that tries to briefly acquire c.mux from
// inside the mock's UnregisterRemoteSKI implementation; if the lock is
// held, the acquisition times out and the test fails.
func TestUnregisterDevice_MutexNotHeldDuringShipCall(t *testing.T) {
	dev := &mockDevice{}
	c := &EEBus{
		log:       util.NewLogger("test"),
		clients:   map[string][]Device{"aabbcc": {dev}},
		connected: make(map[string]bool),
	}

	service := eebusmocks.NewServiceInterface(t)
	service.EXPECT().UnregisterRemoteSKI("aabbcc").Run(func(string) {
		acquired := make(chan struct{})
		go func() {
			c.mux.Lock()
			c.mux.Unlock()
			close(acquired)
		}()
		select {
		case <-acquired:
			// good — mutex was free
		case <-time.After(100 * time.Millisecond):
			t.Errorf("c.mux was held while UnregisterRemoteSKI was called — " +
				"regression to the cross-layer lock hold that caused #28942")
		}
	}).Once()
	c.service = service

	c.UnregisterDevice("aabbcc", dev)
}
