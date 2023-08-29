package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/mock"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSyncCharger(t *testing.T) {
	tc := []struct {
		expected, actual bool
	}{
		{false, false},
		{false, true},
		{true, false},
		{true, true},
	}

	ctrl := gomock.NewController(t)

	for _, tc := range tc {
		charger := mock.NewMockCharger(ctrl)
		charger.EXPECT().Enabled().Return(tc.actual, nil).AnyTimes()

		lp := &Loadpoint{
			log:     util.NewLogger("foo"),
			clock:   clock.New(),
			charger: charger,
			enabled: tc.expected,
		}

		assert.NoError(t, lp.syncCharger())
		assert.Equal(t, tc.actual, lp.enabled)
	}
}
