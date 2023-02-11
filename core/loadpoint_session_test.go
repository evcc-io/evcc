package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	coredb "github.com/evcc-io/evcc/core/db"
	serverdb "github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/assert"
)

func TestSession(t *testing.T) {
	var err error
	serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
	assert.NoError(t, err)

	db, err := coredb.New("foo")
	assert.NoError(t, err)

	clock := clock.NewMock()

	lp := &Loadpoint{
		clock: clock,
		db:    db,
	}

	// create session
	lp.createSession()
	assert.NotNil(t, lp.session)

	// start charging
	lp.updateSession(func(session *coredb.Session) {
		if session.Created.IsZero() {
			session.Created = lp.clock.Now()
		}
	})
	assert.Equal(t, clock.Now(), lp.session.Created)

	// stop charging
	clock.Add(time.Hour)
	lp.chargedEnergy = 1.23

	lp.stopSession()
	assert.NotNil(t, lp.session)
	assert.Equal(t, lp.chargedEnergy/1e3, lp.session.ChargedEnergy)
	assert.Equal(t, clock.Now(), lp.session.Finished)

	s, err := db.Sessions()
	assert.NoError(t, err)
	assert.Len(t, s, 1)
	t.Log(s)

	// stop charging - 2nd leg
	clock.Add(time.Hour)
	lp.chargedEnergy *= 2

	lp.stopSession()
	assert.NotNil(t, lp.session)
	assert.Equal(t, lp.chargedEnergy/1e3, lp.session.ChargedEnergy)
	assert.Equal(t, clock.Now(), lp.session.Finished)

	s, err = db.Sessions()
	assert.NoError(t, err)
	assert.Len(t, s, 1)
	t.Log(s)
}
