package charger

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/require"
)

// kebaHandler mocks the charger's holding register space
type kebaHandler struct {
	mbserver.RequestHandler
	mu   sync.Mutex
	regs map[uint16]uint16
}

func (h *kebaHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.IsWrite {
		for i, v := range req.Args {
			h.regs[req.Addr+uint16(i)] = v
		}
		return req.Args, nil
	}

	res := make([]uint16, 0, req.Quantity)
	for i := range req.Quantity {
		res = append(res, h.regs[req.Addr+i])
	}

	return res, nil
}

// shared mock server: mbserver.Stop() races its accept goroutine, so the server
// is started once and never stopped; handler state is reset per test
var (
	kebaOnce sync.Once
	kebaURI  string
	kebaSrvH = &kebaHandler{RequestHandler: new(mbserver.DummyHandler)}
)

// kebaTestCharger returns a P30 with external phase switching connected to the mock server
func kebaTestCharger(t *testing.T, phases int) (*Keba, *kebaHandler) {
	t.Helper()

	kebaOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		srv, err := mbserver.New(kebaSrvH)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))

		kebaURI = l.Addr().String()
	})

	var state uint16
	if phases == 3 {
		state = 1
	}
	kebaSrvH.regs = map[uint16]uint16{
		kebaRegPhaseState + 1: state,
		kebaRegTriggerPhase:   state,
	}

	conn, err := modbus.NewConnection(context.Background(), kebaURI, "", "", 0, modbus.Tcp, 255)
	require.NoError(t, err)

	wb := &Keba{
		embed:        new(embed),
		Caps:         implement.New(),
		log:          util.NewLogger("keba"),
		conn:         conn,
		regEnable:    kebaRegEnable,
		energyFactor: 1e4,
		phases:       phases,
	}

	return wb, kebaSrvH
}

func (h *kebaHandler) reg(addr uint16) uint16 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.regs[addr]
}

// TestKebaEnablePhases verifies that the external phase switch relay is released
// while disabled and restored on enable, see discussion #32176
func TestKebaEnablePhases(t *testing.T) {
	wb, h := kebaTestCharger(t, 3)

	require.NoError(t, wb.Enable(false))
	require.Equal(t, uint16(0), h.reg(kebaRegTriggerPhase), "relay must be released on disable")
	require.Equal(t, 3, wb.phases, "requested phases must be retained")

	require.NoError(t, wb.Enable(true))
	require.Equal(t, uint16(1), h.reg(kebaRegTriggerPhase), "relay must be restored on enable")

	// a 1p charger stays 1p
	wb, h = kebaTestCharger(t, 1)

	require.NoError(t, wb.Enable(true))
	require.Equal(t, uint16(0), h.reg(kebaRegTriggerPhase))
	require.NoError(t, wb.Enable(false))
	require.Equal(t, uint16(0), h.reg(kebaRegTriggerPhase))
}

// TestKebaEnableWithoutPhaseSwitching verifies the phase register is untouched
// when the charger has no external phase switching
func TestKebaEnableWithoutPhaseSwitching(t *testing.T) {
	wb, h := kebaTestCharger(t, 0)
	h.regs[kebaRegTriggerPhase] = 1

	require.NoError(t, wb.Enable(false))
	require.Equal(t, uint16(1), h.reg(kebaRegTriggerPhase))
}
