package charger

import (
	"net"
	"testing"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/modbus"
	modbusutils "github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvseCheckStatus(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	var handler = modbus.DevicesSimulatorHandler{
		Devices: map[uint8]struct {
			Coils    map[uint16]bool
			Discrete map[uint16]bool
			Input    map[uint16]uint16
			Holding  map[uint16]uint16
		}{
			1: {
				Holding: map[uint16]uint16{
					evseRegFirmware: 0x12, // firmware version 18
					evseRegConfig:   0x92, // config with milliamps enabled
					evseRegCurrent:  0x0,  // current 0A
					evseRegStatus:   0x3,  // statusC: charging
				},
			},
		},
	}
	srv, _ := mbserver.New(&handler)

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	evse, err := NewEvseDIN(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1)
	require.NoError(t, err)

	st, err := evse.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, st)
}

func TestEvseCheckEnableMilliamp(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	var handler = modbus.DevicesSimulatorHandler{
		Devices: map[uint8]struct {
			Coils    map[uint16]bool
			Discrete map[uint16]bool
			Input    map[uint16]uint16
			Holding  map[uint16]uint16
		}{
			1: {
				Holding: map[uint16]uint16{
					evseRegFirmware: 0x12, // firmware version 18
					evseRegConfig:   0x92, // config with milliamps enabled
					evseRegCurrent:  0x0,  // current 0A
					evseRegStatus:   0x3,  // statusC: charging
				},
			},
		},
	}
	srv, _ := mbserver.New(&handler)

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	evse, err := NewEvseDIN(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1)
	require.NoError(t, err)

	// check that charger is initially disabled
	enabled, err := evse.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled, "charger should be disabled initially")

	// Now enable charger and check programmed current value
	require.NoError(t, evse.Enable(true))
	assert.Equal(t, handler.Devices[1].Holding[evseRegCurrent], uint16(600), "current not set to 6.00A after enabling")

	ex, ok := evse.(api.ChargerEx)
	assert.True(t, ok, "missing ChargerEx interface")

	// set milliamps current and check programmed value
	require.NoError(t, ex.MaxCurrentMillis(12.34))
	assert.Equal(t, handler.Devices[1].Holding[evseRegCurrent], uint16(1234), "current not set to 12.34A after MaxCurrentMillis(12.34)")
}

func TestEvseCheckEnableAmp(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	var handler = modbus.DevicesSimulatorHandler{
		Devices: map[uint8]struct {
			Coils    map[uint16]bool
			Discrete map[uint16]bool
			Input    map[uint16]uint16
			Holding  map[uint16]uint16
		}{
			1: {
				Holding: map[uint16]uint16{
					evseRegFirmware: 0x10, // firmware version 16
					evseRegConfig:   0x12, // config without milliamps disabled
					evseRegCurrent:  0x0,  // current 0A
					evseRegStatus:   0x3,  // statusC: charging
				},
			},
		},
	}
	srv, _ := mbserver.New(&handler)

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	evse, err := NewEvseDIN(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1)
	require.NoError(t, err)

	// check that charger is initially disabled
	enabled, err := evse.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled, "charger should be disabled initially")

	// Now enable charger and check programmed current value
	require.NoError(t, evse.Enable(true))
	assert.Equal(t, handler.Devices[1].Holding[evseRegCurrent], uint16(6), "current not set to 6A after enabling")

	// set amps current and check programmed value
	require.NoError(t, evse.MaxCurrent(12))
	assert.Equal(t, handler.Devices[1].Holding[evseRegCurrent], uint16(12), "current not set to 12A after setting MaxCurrent(12)")
}

func TestEvseMilliampsRegulation(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	var handler = modbus.DevicesSimulatorHandler{
		Devices: map[uint8]struct {
			Coils    map[uint16]bool
			Discrete map[uint16]bool
			Input    map[uint16]uint16
			Holding  map[uint16]uint16
		}{
			1: {
				Holding: map[uint16]uint16{
					evseRegFirmware: 0x12, // firmware version 18
					evseRegConfig:   0x92, // config with milliamps enabled
					evseRegCurrent:  0x0,  // current 0A
					evseRegStatus:   0x3,  // statusC: charging
				},
			},
		},
	}
	srv, _ := mbserver.New(&handler)

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	evse, err := NewEvseDIN(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1)
	require.NoError(t, err)

	if _, ok := evse.(api.ChargerEx); !ok {
		assert.Fail(t, "missing milliamps support")
	}
	assert.True(t, handler.Devices[1].Holding[evseRegConfig]&0x80 != 0, "milliamps support not enabled in register")
}

func TestEvseFirmwareDoesNotSupportMillampRegulation(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	var handler = modbus.DevicesSimulatorHandler{
		Devices: map[uint8]struct {
			Coils    map[uint16]bool
			Discrete map[uint16]bool
			Input    map[uint16]uint16
			Holding  map[uint16]uint16
		}{
			1: {
				Holding: map[uint16]uint16{
					evseRegFirmware: 0x10, // firmware version 16
					evseRegConfig:   0x12, // milliamps support disabled
					evseRegCurrent:  0x0,  // current 0A
					evseRegStatus:   0x3,  // statusC: charging
				},
			},
		},
	}
	srv, _ := mbserver.New(&handler)

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	evse, err := NewEvseDIN(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1)
	require.NoError(t, err)

	if _, ok := evse.(api.ChargerEx); ok {
		assert.Fail(t, "milliamps support unexpected")
	}
}

func TestEvseFirmwareDoesSupportMillampRegulationButNotEnabledByDefault(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	var handler = modbus.DevicesSimulatorHandler{
		Devices: map[uint8]struct {
			Coils    map[uint16]bool
			Discrete map[uint16]bool
			Input    map[uint16]uint16
			Holding  map[uint16]uint16
		}{
			1: {
				Holding: map[uint16]uint16{
					evseRegFirmware: 0x12, // firmware version 16
					evseRegConfig:   0x12, // milliamps support disabled
					evseRegCurrent:  0x0,  // current 0A
					evseRegStatus:   0x3,  // statusC: charging
				},
			},
		},
	}

	srv, _ := mbserver.New(&handler)

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	evse, err := NewEvseDIN(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1)
	require.NoError(t, err)

	if _, ok := evse.(api.ChargerEx); !ok {
		assert.Fail(t, "missing milliamps support")
	}
	assert.True(t, handler.Devices[1].Holding[evseRegConfig]&0x80 != 0, "milliamps support not enabled in register")
}
