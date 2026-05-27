package eebus

import (
	"testing"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	eebusmocks "github.com/enbility/eebus-go/mocks"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/util"
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
		log:     util.NewLogger("test"),
		clients: map[string][]Device{"aabbcc": {dev}},
	}

	service := eebusmocks.NewServiceInterface(t)
	service.EXPECT().UnregisterRemoteSKI("aabbcc").Run(func(string) {
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
