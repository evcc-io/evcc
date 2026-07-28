package eebus

import (
	"testing"

	ucmocks "github.com/enbility/eebus-go/usecases/mocks"
	spineapi "github.com/enbility/spine-go/api"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/stretchr/testify/assert"
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

func TestUpdateEntity(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		cachedDepth, entityDepth int
		cachedScenarios          []uint
		entityScenarios          []uint
		want                     string // "cached", "entity" or "nil"
	}{
		{"first entity is cached", 0, 1, nil, []uint{1}, "entity"},
		{"entity without scenarios is ignored", 0, 1, nil, nil, "nil"},
		{"shallower entity wins", 2, 1, []uint{1}, []uint{1}, "entity"},
		{"deeper entity loses", 1, 2, []uint{1}, []uint{1}, "cached"},
		{"removal drops the cached entity", 1, 1, nil, nil, "nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := ucmocks.NewMaMPCInterface(t)

			var cached spineapi.EntityRemoteInterface
			if tc.cachedDepth > 0 {
				cached = testEntity(t, tc.cachedDepth)
				uc.EXPECT().AvailableScenariosForEntity(cached).Return(tc.cachedScenarios).Maybe()
			}

			entity := testEntity(t, tc.entityDepth)
			uc.EXPECT().AvailableScenariosForEntity(entity).Return(tc.entityScenarios).Maybe()

			want := map[string]spineapi.EntityRemoteInterface{
				"cached": cached, "entity": entity, "nil": nil,
			}[tc.want]

			assert.Equal(t, want, UpdateEntity(uc, cached, entity))
		})
	}
}

// removing the use case at another entity must not drop the cached one
func TestUpdateEntityRemovalOfOtherEntity(t *testing.T) {
	uc := ucmocks.NewMaMPCInterface(t)

	cached := testEntity(t, 1)
	uc.EXPECT().AvailableScenariosForEntity(cached).Return([]uint{1})

	removed := testEntity(t, 2)
	uc.EXPECT().AvailableScenariosForEntity(removed).Return(nil)

	assert.Equal(t, cached, UpdateEntity(uc, cached, removed))
}
