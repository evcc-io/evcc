package eebus

import (
	"errors"
	"testing"

	eebusapi "github.com/enbility/eebus-go/api"
	ucmocks "github.com/enbility/eebus-go/usecases/mocks"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadValue(t *testing.T) {
	read := func(res float64, err error) func(spineapi.EntityRemoteInterface) (float64, error) {
		return func(spineapi.EntityRemoteInterface) (float64, error) { return res, err }
	}

	t.Run("entity missing", func(t *testing.T) {
		uc := ucmocks.NewMaMPCInterface(t)

		_, err := ReadValue(uc, MPCPower, nil, read(1, nil))
		assert.ErrorIs(t, err, api.ErrNotAvailable)
	})

	t.Run("scenario not announced", func(t *testing.T) {
		uc := ucmocks.NewMaMPCInterface(t)
		entity := testEntity(t, 1)
		uc.EXPECT().IsScenarioAvailableAtEntity(entity, MPCPower).Return(false)

		_, err := ReadValue(uc, MPCPower, entity, read(1, nil))
		assert.ErrorIs(t, err, api.ErrNotAvailable)
	})

	// data not received yet is not an error condition
	for _, tc := range []error{eebusapi.ErrDataNotAvailable, eebusapi.ErrMetadataNotAvailable, eebusapi.ErrDataInvalid} {
		t.Run(tc.Error(), func(t *testing.T) {
			uc := ucmocks.NewMaMPCInterface(t)
			entity := testEntity(t, 1)
			uc.EXPECT().IsScenarioAvailableAtEntity(entity, MPCPower).Return(true)

			_, err := ReadValue(uc, MPCPower, entity, read(0, tc))
			assert.ErrorIs(t, err, api.ErrNotAvailable)
		})
	}

	t.Run("other error passes through", func(t *testing.T) {
		uc := ucmocks.NewMaMPCInterface(t)
		entity := testEntity(t, 1)
		uc.EXPECT().IsScenarioAvailableAtEntity(entity, MPCPower).Return(true)

		_, err := ReadValue(uc, MPCPower, entity, read(0, errors.New("boom")))
		assert.EqualError(t, err, "boom")
	})

	t.Run("value", func(t *testing.T) {
		uc := ucmocks.NewMaMPCInterface(t)
		entity := testEntity(t, 1)
		uc.EXPECT().IsScenarioAvailableAtEntity(entity, MPCPower).Return(true)

		res, err := ReadValue(uc, MPCPower, entity, read(4711, nil))
		require.NoError(t, err)
		assert.Equal(t, 4711.0, res)
	})
}
