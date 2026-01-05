package charger

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/modbus"
	modbusutils "github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define gpio mocks for testing
type mockGpioPin struct {
	pin int
	ops *[]string
}

func (m mockGpioPin) High() { *m.ops = append(*m.ops, fmt.Sprintf("Setting pin %d to high", m.pin)) }
func (m mockGpioPin) Low()  { *m.ops = append(*m.ops, fmt.Sprintf("Setting pin %d to low", m.pin)) }
func (m mockGpioPin) Output() {
	*m.ops = append(*m.ops, fmt.Sprintf("Setting pin %d to output", m.pin))
}

type mockGpio struct {
	opened bool
	pins   map[int]mockGpioPin
	ops    []string
}

func (m *mockGpio) Open() error {
	m.opened = true
	if m.pins == nil {
		m.pins = make(map[int]mockGpioPin)
	}
	m.ops = make([]string, 0)
	return nil
}

func (m *mockGpio) Close() {
	m.opened = false
}

func (m *mockGpio) Pin(p int) gpioPin {
	if m.pins == nil {
		m.pins = make(map[int]mockGpioPin)
	}
	if _, ok := m.pins[p]; !ok {
		m.pins[p] = mockGpioPin{pin: p, ops: &m.ops}
	}
	return m.pins[p]
}

func TestOpenwbNativeMilliampsRegulation(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(&modbus.DevicesSimulatorHandler{
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
	})

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	wb, err := NewOpenWbNative(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1, true, "", 0, 1, &mockGpio{})
	require.NoError(t, err)

	if _, ok := wb.(api.ChargerEx); !ok {
		assert.Fail(t, "missing milliamp support")
	}
}

func TestOpenwbNativeFirmwareDoesNotSupportMillampRegulation(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(&modbus.DevicesSimulatorHandler{
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
	})

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	wb, err := NewOpenWbNative(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1, true, "", 0, 1, &mockGpio{})
	require.NoError(t, err)

	if _, ok := wb.(api.ChargerEx); ok {
		assert.Fail(t, "milliamp support unexpected")
	}
}

func TestGpios(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(&modbus.DevicesSimulatorHandler{
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
	})

	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	gpio := &mockGpio{}
	wb, err := NewOpenWbNative(t.Context(), l.Addr().String(), "", "", 0, modbusutils.Tcp, 1, true, "", 0*time.Second, 1, gpio)
	require.NoError(t, err)

	// Check initialization of GPIO pins to output happened
	assert.Equal(t, []string{"Setting pin 25 to output", "Setting pin 5 to output", "Setting pin 26 to output"}, gpio.ops, "GPIO setup wrong!")

	if _, ok := wb.(api.PhaseSwitcher); !ok {
		assert.Fail(t, "missing phase switch support")
	}

	ps := wb.(api.PhaseSwitcher)
	require.NoError(t, ps.Phases1p3p(3))
	assert.Equal(t, []string{"Setting pin 25 to high", "Setting pin 26 to high", "Setting pin 26 to low", "Setting pin 25 to low"}, gpio.ops, "GPIO pin setting for three phases is incorrect!")

	require.NoError(t, ps.Phases1p3p(1))
	assert.Equal(t, []string{"Setting pin 25 to high", "Setting pin 5 to high", "Setting pin 5 to low", "Setting pin 25 to low"}, gpio.ops, "GPIO pin setting for one phase is incorrect!")

	if _, ok := wb.(api.Resurrector); !ok {
		assert.Fail(t, "missing resurrector support")
	}

	rs := wb.(api.Resurrector)
	require.NoError(t, rs.WakeUp())
	assert.Equal(t, []string{"Setting pin 25 to high", "Setting pin 25 to low"}, gpio.ops, "GPIO pin setting for wakeup is incorrect!")
}
