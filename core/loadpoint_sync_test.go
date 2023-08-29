package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/mock"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSyncCharger(t *testing.T) {
	tc := []struct {
		status                      api.ChargeStatus
		expected, actual, corrected bool
	}{
		{api.StatusA, false, false, false},
		{api.StatusC, false, false, true}, // disabled but charging
		{api.StatusA, false, true, true},
		{api.StatusA, true, false, false},
		{api.StatusA, true, true, true},
	}

	ctrl := gomock.NewController(t)

	for _, tc := range tc {
		t.Logf("%+v", tc)

		charger := mock.NewMockCharger(ctrl)
		charger.EXPECT().Enabled().Return(tc.actual, nil).AnyTimes()

		lp := &Loadpoint{
			log:     util.NewLogger("foo"),
			clock:   clock.New(),
			charger: charger,
			status:  tc.status,
			enabled: tc.expected,
		}

		assert.NoError(t, lp.syncCharger())
		assert.Equal(t, tc.corrected, lp.enabled)
	}
}
