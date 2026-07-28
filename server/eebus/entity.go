package eebus

import (
	"sync"

	eebusapi "github.com/enbility/eebus-go/api"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/evcc-io/evcc/api"
)

// Entity is the remote entity a use case is available at. It knows its use case
// and guards its own access.
type Entity[U eebusapi.UseCaseBaseInterface] struct {
	uc U

	mu     sync.RWMutex
	entity spineapi.EntityRemoteInterface
}

// NewEntity creates the remote entity of a use case
func NewEntity[U eebusapi.UseCaseBaseInterface](uc U) *Entity[U] {
	return &Entity[U]{uc: uc}
}

// Get returns the remote entity, nil if the use case has none (yet)
func (e *Entity[U]) Get() spineapi.EntityRemoteInterface {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.entity
}

// Set records the remote entity, use nil to drop it e.g. on disconnect
func (e *Entity[U]) Set(entity spineapi.EntityRemoteInterface) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.entity = entity
}

// Update records the remote entity of a use case support update. Removal emits the
// same event with the entity set, so it is dropped once its scenarios are gone. It
// reports whether the use case became available at a new entity.
func (e *Entity[U]) Update(entity spineapi.EntityRemoteInterface) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	prev := e.entity

	if e.entity != nil && len(e.uc.AvailableScenariosForEntity(e.entity)) == 0 {
		e.entity = nil
	}

	// removal, or an entity that doesn't support any scenario of the use case
	if entity == nil || len(e.uc.AvailableScenariosForEntity(entity)) == 0 {
		return false
	}

	if e.entity == nil || len(entity.Address().Entity) < len(e.entity.Address().Entity) {
		e.entity = entity
	}

	return e.entity != nil && e.entity != prev
}

// Read reads a use case value from the remote entity, reporting ErrNotAvailable
// while the scenario is unavailable or the value has not been received yet.
func (e *Entity[U]) Read[T any](scenario uint, read func(uc U, entity spineapi.EntityRemoteInterface) (T, error)) (T, error) {
	var zero T

	entity := e.Get()
	if entity == nil || !e.uc.IsScenarioAvailableAtEntity(entity, scenario) {
		return zero, api.ErrNotAvailable
	}

	res, err := read(e.uc, entity)
	if err != nil {
		return zero, WrapError(err)
	}

	return res, nil
}

// Required returns the remote entity while the use case scenario is available at
// it, ErrNotConnected otherwise. Optional data uses Read.
func (e *Entity[U]) Required(scenario uint) (spineapi.EntityRemoteInterface, error) {
	entity := e.Get()

	if entity == nil || !e.uc.IsScenarioAvailableAtEntity(entity, scenario) {
		return nil, ErrNotConnected
	}

	return entity, nil
}
