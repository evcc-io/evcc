package ocpp

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterChargepointClearsCacheOnInitFailure(t *testing.T) {
	cs := &CS{
		log:  util.NewLogger("foo"),
		regs: make(map[string]*registration),
	}

	newfun := func() *CP { return NewChargePoint(util.NewLogger("foo"), cs, "test") }

	// first attempt fails- must not leave a stale cp cached
	_, err := cs.RegisterChargepoint("test", newfun, func(*CP) error {
		return errors.New("boom")
	})
	require.Error(t, err)
	assert.Nil(t, cs.regs["test"].cp, "failed init must not cache the charge point")

	// retry must create a fresh charge point and succeed
	var initialised *CP
	cp, err := cs.RegisterChargepoint("test", newfun, func(cp *CP) error {
		initialised = cp
		return nil
	})
	require.NoError(t, err)
	assert.Same(t, initialised, cp)
	assert.Same(t, cp, cs.regs["test"].cp, "successful init must cache the charge point")

	// further calls must return the cached charge point without re-running init
	cp2, err := cs.RegisterChargepoint("test", newfun, func(*CP) error {
		t.Fatal("init must not be called again for an already registered charge point")
		return nil
	})
	require.NoError(t, err)
	assert.Same(t, cp, cp2)
}
