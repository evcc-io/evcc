package eebus

import (
	"testing"

	ucmocks "github.com/enbility/eebus-go/usecases/mocks"
	spineapi "github.com/enbility/spine-go/api"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEntity returns a remote entity mock with an address of the given depth
func testEntity(t *testing.T, depth int) spineapi.EntityRemoteInterface {
	t.Helper()

	entity := spinemocks.NewEntityRemoteInterface(t)
	entity.EXPECT().Address().Return(&model.EntityAddressType{
		Entity: make([]model.AddressEntityType, depth),
	}).Maybe()

	return entity
}

func TestEntityUpdate(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		cachedDepth, entityDepth int
		cachedScenarios          []uint
		entityScenarios          []uint
		want                     string // "cached", "entity" or "nil"
	}{
		{"first entity is recorded", 0, 1, nil, []uint{1}, "entity"},
		{"entity without scenarios is ignored", 0, 1, nil, nil, "nil"},
		{"shallower entity wins", 2, 1, []uint{1}, []uint{1}, "entity"},
		{"deeper entity loses", 1, 2, []uint{1}, []uint{1}, "cached"},
		{"removal drops the recorded entity", 1, 1, nil, nil, "nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := ucmocks.NewMaMPCInterface(t)

			var e Entity
			var cached spineapi.EntityRemoteInterface
			if tc.cachedDepth > 0 {
				cached = testEntity(t, tc.cachedDepth)
				uc.EXPECT().AvailableScenariosForEntity(cached).Return(tc.cachedScenarios).Maybe()
				e.Set(cached)
			}

			entity := testEntity(t, tc.entityDepth)
			uc.EXPECT().AvailableScenariosForEntity(entity).Return(tc.entityScenarios).Maybe()

			want := map[string]spineapi.EntityRemoteInterface{
				"cached": cached, "entity": entity, "nil": nil,
			}[tc.want]

			e.Update(uc, entity)
			assert.Equal(t, want, e.Get())
		})
	}
}

// removing the use case at another entity must not drop the recorded one
func TestEntityUpdateRemovalOfOtherEntity(t *testing.T) {
	uc := ucmocks.NewMaMPCInterface(t)

	cached := testEntity(t, 1)
	uc.EXPECT().AvailableScenariosForEntity(cached).Return([]uint{1})

	removed := testEntity(t, 2)
	uc.EXPECT().AvailableScenariosForEntity(removed).Return(nil)

	var e Entity
	e.Set(cached)

	e.Update(uc, removed)
	assert.Equal(t, cached, e.Get())
}

func TestEntityRequired(t *testing.T) {
	t.Run("entity missing", func(t *testing.T) {
		uc := ucmocks.NewMaMPCInterface(t)

		var e Entity
		_, err := e.Required(uc, MPCPower)
		assert.ErrorIs(t, err, ErrNotConnected)
	})

	t.Run("scenario not announced", func(t *testing.T) {
		uc := ucmocks.NewMaMPCInterface(t)
		entity := testEntity(t, 1)
		uc.EXPECT().IsScenarioAvailableAtEntity(entity, MPCPower).Return(false)

		var e Entity
		e.Set(entity)

		_, err := e.Required(uc, MPCPower)
		assert.ErrorIs(t, err, ErrNotConnected)
	})

	t.Run("available", func(t *testing.T) {
		uc := ucmocks.NewMaMPCInterface(t)
		entity := testEntity(t, 1)
		uc.EXPECT().IsScenarioAvailableAtEntity(entity, MPCPower).Return(true)

		var e Entity
		e.Set(entity)

		res, err := e.Required(uc, MPCPower)
		require.NoError(t, err)
		assert.Equal(t, entity, res)
	})
}
