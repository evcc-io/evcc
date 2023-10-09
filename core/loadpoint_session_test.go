package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/mock"
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

	mm := mock.NewMockMeter(ctrl)
	me := mock.NewMockMeterEnergy(ctrl)

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

func TestCloseSessionsOnStartup(t *testing.T) {
	var err error
	serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
	assert.NoError(t, err)

	db, err := session.NewStore("foo", serverdb.Instance)
	assert.NoError(t, err)

	clock := clock.NewMock()

	//test data, creates 6 sessions, 3rd and 6th are "unfinished"
	var meterStart float64
	for i := 1; i <= 6; i++ {
		meterStart += 10
		session := db.New(meterStart)
		session.Created = clock.Now().Add(1 * time.Minute)
		if i%3 == 0 { //create every third session as incomplete
			db.Persist(session)
			continue
		}

		session.Finished = clock.Now().Add(2 * time.Minute)
		meterStop := float64(meterStart + 10)
		session.MeterStop = &meterStop
		session.ChargedEnergy = 10
		db.Persist(session)
	}

	err = db.ClosePendingSessionsInHistory()
	assert.NoError(t, err)

	var ss session.Sessions
	err = serverdb.Instance.Order("ID").Find(&ss).Error
	assert.NoError(t, err)
	assert.Len(t, ss, 6)

	//check fixed history
	for _, s := range ss[:5] {
		assert.NotEmpty(t, s.MeterStop)
		assert.Equal(t, float64(10), s.ChargedEnergy)
		t.Logf("session: %+v", s)
	}

	// cannot fix the last session, which has no successor yet, ensure it was left alone
	s := ss[5]
	assert.Empty(t, s.MeterStop)
	assert.Empty(t, s.ChargedEnergy)
}
