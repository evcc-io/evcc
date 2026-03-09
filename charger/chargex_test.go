package charger

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chargexHandler is a minimal Modbus server for ChargeX tests.
// It records every write to PAC_Target_Power (reg 504).
type chargexHandler struct {
	mbserver.DummyHandler
	mu           sync.Mutex
	targetWrites []uint32
	timeout      uint32 // returned for PAC_Target_Timeout (reg 500), in seconds
	moduleState  uint32 // returned for States_CP (reg 108 for connector 1)
}

func (h *chargexHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.IsWrite {
		if req.Addr == chargexRegTargetPower {
			h.targetWrites = append(h.targetWrites, uint32(req.Args[0])<<16|uint32(req.Args[1]))
		}
		return nil, nil
	}

	switch req.Addr {
	case chargexRegTargetTimeout:
		return []uint16{uint16(h.timeout >> 16), uint16(h.timeout)}, nil
	case chargexRegTargetPower:
		var last uint32
		if len(h.targetWrites) > 0 {
			last = h.targetWrites[len(h.targetWrites)-1]
		}
		return []uint16{uint16(last >> 16), uint16(last)}, nil
	case 108: // moduleReg(connector=1, chargexRegModuleState): 100 + 0*12 + 8
		return []uint16{uint16(h.moduleState >> 16), uint16(h.moduleState)}, nil
	}

	return []uint16{0, 0}, nil
}

// TestChargeXHeartbeat verifies that the ChargeX driver periodically
// re-sends the target power to prevent the charger's PAC_Target_Timeout
// from expiring and triggering a fallback to PAC_Default_Power.
func TestChargeXHeartbeat(t *testing.T) {
	sponsor.Subject = "test"

	h := &chargexHandler{
		timeout:     2,      // 2 s timeout → 1 s heartbeat interval
		moduleState: 1 << 1, // bit 1: 3-phase
	}

	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(h)
	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	wb, err := NewChargeX(t.Context(), l.Addr().String(), 10, 1)
	require.NoError(t, err)

	require.NoError(t, wb.Enable(false))

	// Expect at least 3 writes: the initial Enable(false) write plus
	// at least 2 heartbeat ticks at 1 s intervals.
	require.Eventually(t, func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return len(h.targetWrites) >= 3
	}, 10*time.Second, 100*time.Millisecond)

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, w := range h.targetWrites {
		assert.Equal(t, uint32(0), w, "heartbeat must keep target power at 0 W while disabled")
	}
}
