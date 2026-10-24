package ecoflow

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ecoflow/ecoflowtest"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestWallbox answers every write with a heartbeat carrying state and phase mode field 21.
func newTestWallbox(t *testing.T, state, phaseMode uint64) *Wallbox {
	t.Helper()

	transport := ecoflowtest.NewTransport()
	conn := NewConnection(util.NewLogger("test"), transport, testUserID)
	wb, err := conn.Wallbox(ecoflowtest.Serial, ecoflowtest.Gateway, time.Minute)
	require.NoError(t, err)

	transport.OnPublish(func(m ecoflowtest.Message) {
		if m.Topic != "/app/"+testUserID+"/"+ecoflowtest.Gateway+"/thing/property/set" {
			return
		}
		wb.handleHeartbeat(varint(varint(nil, 1, state), 21, phaseMode))
	})

	return wb
}

// After a written 3p the wallbox reports field 21 = 0 (hardware, 2026-10-24).
func TestSetPhaseModeThreeConfirmedByField21(t *testing.T) {
	wb := newTestWallbox(t, uint64(StateStoppedByWallbox), 0)
	assert.NoError(t, wb.SetPhaseMode(PhaseThree, 50*time.Millisecond))

	wb = newTestWallbox(t, uint64(StateStoppedByWallbox), 1)
	assert.ErrorIs(t, wb.SetPhaseMode(PhaseThree, 50*time.Millisecond), api.ErrTimeout)
}

func TestWriteWithoutGateway(t *testing.T) {
	conn := NewConnection(util.NewLogger("test"), ecoflowtest.NewTransport(), testUserID)
	wb, err := conn.Wallbox(ecoflowtest.Serial, "", time.Minute)
	require.NoError(t, err)

	assert.Error(t, wb.SetMaxCurrent(100, 50*time.Millisecond))
}
