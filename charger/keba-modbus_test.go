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

func (h *kebaHandler) reg(addr uint16) uint16 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.regs[addr]
}

// shared mock server: mbserver.Stop() races its accept goroutine, so the server
// is started once and never stopped; handler state is reset per test
var (
	kebaOnce sync.Once
	kebaURI  string
	kebaSrvH = &kebaHandler{RequestHandler: new(mbserver.DummyHandler)}
)

// kebaTestCharger returns a P30 connected to the mock server, charging on the given
// number of phases and optionally offering external phase switching
func kebaTestCharger(t *testing.T, phases int, switchable bool) (*Keba, *kebaHandler) {
	t.Helper()

	kebaOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		srv, err := mbserver.New(kebaSrvH)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))

		kebaURI = l.Addr().String()
	})

	// P30 reports 3p as 1 in the upper phase state register
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
	}

	if switchable {
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))
	}

	return wb, kebaSrvH
}

// TestKebaEnablePhases verifies that the external phase switch relay is released
// on disable, see discussion #32176
func TestKebaEnablePhases(t *testing.T) {
	tc := []struct {
		name       string
		phases     int
		switchable bool
		enable     bool
		expected   uint16
	}{
		{"3p disable releases relay", 3, true, false, 0},
		{"3p enable keeps relay", 3, true, true, 1},
		{"1p disable is a no-op", 1, true, false, 0},
		{"3p without phase switching is untouched", 3, false, false, 1},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			wb, h := kebaTestCharger(t, tc.phases, tc.switchable)

			require.NoError(t, wb.Enable(tc.enable))
			require.Equal(t, tc.expected, h.reg(kebaRegTriggerPhase))
		})
	}
}
