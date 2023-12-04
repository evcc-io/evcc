package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	serverdb "github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSession(t *testing.T) {
	var err error
	serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
	assert.NoError(t, err)

	db, err := session.NewStore("foo", serverdb.Instance)
	assert.NoError(t, err)

	clock := clock.NewMock()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mm := api.NewMockMeter(ctrl)
	me := api.NewMockMeterEnergy(ctrl)

	type EnergyDecorator struct {
		api.Meter
		api.MeterEnergy
	}

	cm := &EnergyDecorator{Meter: mm, MeterEnergy: me}

	lp := &Loadpoint{
		log:           util.NewLogger("foo"),
		clock:         clock,
		db:            db,
		chargeMeter:   cm,
		sessionEnergy: NewEnergyMetrics(),
	}

	// create session
	me.EXPECT().TotalEnergy().Return(1.0, nil)
	lp.createSession()
	assert.NotNil(t, lp.session)

	// start charging
	lp.updateSession(func(session *session.Session) {
		if session.Created.IsZero() {
			session.Created = lp.clock.Now()
		}
	})
	assert.Equal(t, clock.Now(), lp.session.Created)

	// stop charging
	clock.Add(time.Hour)
	lp.sessionEnergy.Update(1.23)
	me.EXPECT().TotalEnergy().Return(1.0+lp.getChargedEnergy()/1e3, nil) // match chargedEnergy

	lp.stopSession()
	assert.NotNil(t, lp.session)
	assert.Equal(t, lp.getChargedEnergy()/1e3, lp.session.ChargedEnergy)
	assert.Equal(t, clock.Now(), lp.session.Finished)

	s, err := db.Sessions()
	assert.NoError(t, err)
	assert.Len(t, s, 1)
	t.Logf("session: %+v", s)

	// stop charging - 2nd leg
	clock.Add(time.Hour)
	lp.sessionEnergy.Update(lp.getChargedEnergy() * 2)
	me.EXPECT().TotalEnergy().Return(3.0, nil) // doesn't match chargedEnergy

	lp.stopSession()
	assert.NotNil(t, lp.session)
	assert.Equal(t, clock.Now(), lp.session.Finished)

	s, err = db.Sessions()
	assert.NoError(t, err)
	assert.Len(t, s, 1)
	t.Logf("session: %+v", s)
}

func TestCloseSessionsOnStartup_emptyDb(t *testing.T) {
	var err error
	serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
	assert.NoError(t, err)

	db, err := session.NewStore("foo", serverdb.Instance)
	assert.NoError(t, err)

	// assert empty DB is no problem
	err = db.ClosePendingSessionsInHistory(1000)
	assert.NoError(t, err)
}

func TestCloseSessionsOnStartup(t *testing.T) {
	var err error
	serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
	assert.NoError(t, err)

	db1, err := session.NewStore("foo", serverdb.Instance)
	assert.NoError(t, err)

	db2, err := session.NewStore("bar", serverdb.Instance)
	assert.NoError(t, err)

	clock := clock.NewMock()

	// test data, creates 6 sessions for each loadpoint, 3rd and 6th are "unfinished"
	var sessions1 []*session.Session = createMockSessions(db1, clock)
	var sessions2 []*session.Session = createMockSessions(db2, clock)

	// write interleaved for two loadpoints
	for index, session := range sessions1 {
		db1.Persist(session)
		db2.Persist(sessions2[index])
	}

	err = db1.ClosePendingSessionsInHistory(1000)
	assert.NoError(t, err)

	// check fixed sessions for db1
	var db1Sessions session.Sessions
	err = serverdb.Instance.Where("Loadpoint = ?", "foo").Order("ID").Find(&db1Sessions).Error
	assert.NoError(t, err)
	assert.Len(t, db1Sessions, 6)

	// check fixed history
	for _, s := range db1Sessions[:5] {
		assert.NotEmpty(t, s.MeterStop)
		assert.Equal(t, float64(10), s.ChargedEnergy)
		t.Logf("session: %+v", s)
	}

	// check fixed most recent record
	assert.NotEmpty(t, db1Sessions[5].MeterStop)
	assert.Equal(t, float64(940), db1Sessions[5].ChargedEnergy)

	// ensure no side effects on loadpoint 2 data, i.e. data left unfixed
	var db2Sessions session.Sessions
	err = serverdb.Instance.Where("Loadpoint = ?", "bar").Order("ID").Find(&db2Sessions).Error
	assert.NoError(t, err)
	assert.Len(t, db2Sessions, 6)

	for i, s := range db2Sessions {
		if (i+1)%3 == 0 {
			assert.Empty(t, s.MeterStop)
			assert.Empty(t, s.ChargedEnergy)
			continue
		}
		assert.NotEmpty(t, s.MeterStop)
		assert.Equal(t, float64(10), s.ChargedEnergy)
	}
}

func createMockSessions(db *session.DB, clock *clock.Mock) []*session.Session {
	var sessions []*session.Session
	for i := 1; i <= 6; i++ {

		var meter1Start float64 = float64(i * 10)
		session := db.New(meter1Start)
		session.Created = clock.Now().Add(1 * time.Minute)

		// create every third session as incomplete
		if i%3 == 0 {
			sessions = append(sessions, session)
			continue
		}

		session.Finished = clock.Now().Add(2 * time.Minute)
		meterStop := float64(meter1Start + 10)
		session.MeterStop = &meterStop
		session.ChargedEnergy = 10
		sessions = append(sessions, session)
	}
	return sessions
}
