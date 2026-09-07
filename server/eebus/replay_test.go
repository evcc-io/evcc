package eebus

import (
	"testing"

	eebusapi "github.com/enbility/eebus-go/api"
	eebusmocks "github.com/enbility/eebus-go/mocks"
	"github.com/enbility/eebus-go/usecases/cem/ohpcf"
	shipapi "github.com/enbility/ship-go/api"
	spineapi "github.com/enbility/spine-go/api"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingDevice records the use case events it receives
type recordingDevice struct {
	mockDevice
	events []eebusapi.EventType
}

func (d *recordingDevice) UseCaseEvent(_ spineapi.DeviceRemoteInterface, _ spineapi.EntityRemoteInterface, event eebusapi.EventType) {
	d.events = append(d.events, event)
}

// remoteEntity returns an entity mock belonging to a remote device with the given ski
func remoteEntity(t *testing.T, ski string) spineapi.EntityRemoteInterface {
	t.Helper()

	remote := spinemocks.NewDeviceRemoteInterface(t)
	remote.EXPECT().Ski().Return(ski).Maybe()

	entity := spinemocks.NewEntityRemoteInterface(t)
	entity.EXPECT().Device().Return(remote).Maybe()

	return entity
}

// registering a second device for an already connected ski must re-state the use
// case support, otherwise the config UI device test reports "not connected"
func TestRegisterDeviceReplaysUseCases(t *testing.T) {
	const ski = "aabbcc"

	uc := eebusmocks.NewUseCaseInterface(t)
	uc.EXPECT().RemoteEntitiesScenarios().Return([]eebusapi.RemoteEntityScenarios{
		{Entity: remoteEntity(t, ski)},
		{Entity: remoteEntity(t, "ddeeff")}, // other device, must not be replayed
	}).Once()

	service := eebusmocks.NewServiceInterface(t)
	service.EXPECT().RegisterRemoteService(shipapi.NewServiceIdentity(ski, "", "")).Once()

	c := &EEBus{
		log:       util.NewLogger("test"),
		service:   service,
		clients:   make(map[string][]Device),
		connected: map[string]bool{ski: true},
		useCases:  []useCase{{uc, ohpcf.UseCaseSupportUpdate}},
	}

	dev := new(recordingDevice)
	require.NoError(t, c.RegisterDevice(ski, "", dev))

	assert.Equal(t, []eebusapi.EventType{ohpcf.UseCaseSupportUpdate}, dev.events)
}

// the device paired via the SHIP Pairing Service registers without ski, so the
// replay must resolve the remote through the pairing list
func TestRegisterDevicePairedReplaysUseCases(t *testing.T) {
	const ski = "aabbcc"

	uc := eebusmocks.NewUseCaseInterface(t)
	uc.EXPECT().RemoteEntitiesScenarios().Return([]eebusapi.RemoteEntityScenarios{
		{Entity: remoteEntity(t, ski)},
		{Entity: remoteEntity(t, "ddeeff")}, // connected, but not paired
	}).Once()

	c := &EEBus{
		log:       util.NewLogger("test"),
		ski:       "hostski", // the empty registration ski must not match the host
		service:   eebusmocks.NewServiceInterface(t),
		clients:   make(map[string][]Device),
		connected: map[string]bool{ski: true, "ddeeff": true},
		paired:    []shipapi.ServiceIdentity{shipapi.NewServiceIdentity(ski, "", "")},
		useCases:  []useCase{{uc, ohpcf.UseCaseSupportUpdate}},
	}

	dev := new(recordingDevice)
	require.NoError(t, c.RegisterDevice("", "", dev))

	assert.Equal(t, []eebusapi.EventType{ohpcf.UseCaseSupportUpdate}, dev.events)
}

// a device registering for a ski that is not connected yet gets its use case
// state from the regular event flow, replaying stale entities would be wrong
func TestRegisterDeviceDisconnectedNoReplay(t *testing.T) {
	const ski = "aabbcc"

	service := eebusmocks.NewServiceInterface(t)
	service.EXPECT().RegisterRemoteService(shipapi.NewServiceIdentity(ski, "", "")).Once()

	c := &EEBus{
		log:       util.NewLogger("test"),
		service:   service,
		clients:   make(map[string][]Device),
		connected: make(map[string]bool),
		useCases:  []useCase{{eebusmocks.NewUseCaseInterface(t), ohpcf.UseCaseSupportUpdate}},
	}

	dev := new(recordingDevice)
	require.NoError(t, c.RegisterDevice(ski, "", dev))

	assert.Empty(t, dev.events)
}
