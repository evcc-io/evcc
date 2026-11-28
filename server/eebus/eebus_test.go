package eebus

import (
	"sync"
	"testing"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	eebusmocks "github.com/enbility/eebus-go/mocks"
	shipapi "github.com/enbility/ship-go/api"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestConfig(t *testing.T) {
	conf := `
certificate:
  private: |
    -----BEGIN EC PRIVATE KEY-----
    MHcCfoo==
    -----END EC PRIVATE KEY-----
  public: |
    -----BEGIN CERTIFICATE-----
    MIIBbar=
    -----END CERTIFICATE-----
`

	var res Config
	require.NoError(t, yaml.Unmarshal([]byte(conf), &res))
}

// mockDevice implements Device for testing
type mockDevice struct{}

func (d *mockDevice) Connect(connected bool) {}
func (d *mockDevice) UseCaseEvent(_ spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event eebusapi.EventType) {
}

var _ Device = (*mockDevice)(nil)

// TestUnregisterDevice_MutexNotHeldDuringShipCall is the regression guard
// for issue #28942. It asserts that c.mux is NOT held at the point
// UnregisterRemoteService is called. The pre-fix code held c.mux across that
// cross-layer call, and ship-go's synchronous HandleConnectionClosed
// callback chain re-entered connect(ski, false) on the same goroutine,
// which then deadlocked on c.mux.Lock() (Go mutexes are non-reentrant).
//
// The assertion uses a goroutine that tries to briefly acquire c.mux from
// inside the mock's UnregisterRemoteService implementation; if the lock is
// held, the acquisition times out and the test fails.
func TestUnregisterDevice_MutexNotHeldDuringShipCall(t *testing.T) {
	dev := &mockDevice{}
	c := &EEBus{
		log:     util.NewLogger("test"),
		clients: map[string][]Device{"aabbcc": {dev}},
	}

	service := eebusmocks.NewServiceInterface(t)
	service.EXPECT().UnregisterRemoteService(shipapi.NewServiceIdentity("aabbcc", "", "")).Run(func(shipapi.ServiceIdentity) {
		acquired := make(chan struct{})
		go func() {
			c.mux.Lock()
			defer c.mux.Unlock()
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

// TestPairingDeniedOnlyWhenConfigured guards that an unknown ski is not denied
// while device configuration is still running- its device may register the ski
// moments later, and a denial locks the remote service out until the next restart.
func TestPairingDeniedOnlyWhenConfigured(t *testing.T) {
	identity := shipapi.NewServiceIdentity("aabbcc", "", "")
	detail := shipapi.NewConnectionStateDetail(shipapi.ConnectionStateReceivedPairingRequest, nil)

	service := eebusmocks.NewServiceInterface(t)
	c := &EEBus{
		log:     util.NewLogger("test"),
		clients: make(map[string][]Device),
		pending: make(map[string]shipapi.ServiceIdentity),
		service: service,
	}

	// still configuring - no CancelPairing expected
	c.ServicePairingDetailUpdate(identity, detail)
	require.Len(t, c.pending, 1)

	service.EXPECT().CancelPairing(identity).Once()
	c.configured = true
	c.ServicePairingDetailUpdate(identity, detail)
}

// TestConfigCompleteResolvesPending guards that a request left pending during
// configuration is denied afterwards unless its ski belongs to a configured device-
// ship-go keeps prolonging the pending handshake until somebody decides.
func TestConfigCompleteResolvesPending(t *testing.T) {
	identity := shipapi.NewServiceIdentity("aabbcc", "", "")

	newInstance := func(t *testing.T, clients map[string][]Device) *eebusmocks.ServiceInterface {
		service := eebusmocks.NewServiceInterface(t)
		instance = &EEBus{
			log:     util.NewLogger("test"),
			clients: clients,
			pending: map[string]shipapi.ServiceIdentity{identity.SKI: identity},
			service: service,
		}
		t.Cleanup(func() { instance = nil })
		return service
	}

	t.Run("unknown ski denied", func(t *testing.T) {
		service := newInstance(t, make(map[string][]Device))
		service.EXPECT().CancelPairing(identity).Once()

		ConfigComplete()
		require.True(t, instance.configured)
		require.Empty(t, instance.pending)
	})

	t.Run("configured ski kept", func(t *testing.T) {
		// no CancelPairing expected- the device registered its ski meanwhile
		newInstance(t, map[string][]Device{identity.SKI: {&mockDevice{}}})

		ConfigComplete()
		require.True(t, instance.configured)
	})
}

// TestRegisterDevice_SerializedWithCleanup guards that removing the last device of a ski
// does not revoke the trust of a device registering for that ski at the same time.
func TestRegisterDevice_SerializedWithCleanup(t *testing.T) {
	const ski = "aabbcc"
	identity := shipapi.NewServiceIdentity(ski, "", "")

	for _, tc := range []struct {
		name  string
		block string // SHIP call held until the other operation had the chance to run
	}{
		{"registration during cleanup", "unregister"},
		{"cleanup during registration", "register"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old, replacement := &mockDevice{}, &mockDevice{}

			c := &EEBus{
				log:     util.NewLogger("test"),
				clients: map[string][]Device{ski: {old}},
			}

			var (
				mu    sync.Mutex
				trust []string // effective SHIP trust changes in order
			)
			record := func(s string) {
				mu.Lock()
				defer mu.Unlock()
				trust = append(trust, s)
			}

			entered, release := make(chan struct{}), make(chan struct{})
			hold := func() {
				close(entered)
				<-release
			}

			service := eebusmocks.NewServiceInterface(t)
			service.EXPECT().RegisterRemoteService(identity).Run(func(shipapi.ServiceIdentity) {
				record("register")
				if tc.block == "register" {
					hold()
				}
			}).Once()
			service.EXPECT().UnregisterRemoteService(identity).Run(func(shipapi.ServiceIdentity) {
				if tc.block == "unregister" {
					hold()
				}
				record("unregister")
			}).Maybe()
			c.service = service

			register := func() { require.NoError(t, c.RegisterDevice(ski, "", replacement)) }
			unregister := func() { c.UnregisterDevice(ski, old) }

			first, second := unregister, register
			if tc.block == "register" {
				first, second = register, unregister
			}

			var wg sync.WaitGroup
			wg.Go(first)
			<-entered

			done := make(chan struct{})
			wg.Go(func() {
				second()
				close(done)
			})
			select {
			case <-done:
			case <-time.After(100 * time.Millisecond):
			}

			close(release)
			wg.Wait()

			assert.Equal(t, []Device{replacement}, c.clients[ski])
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, "register", trust[len(trust)-1], "trust changes: %v", trust)
		})
	}
}
