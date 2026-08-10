package eebus

import (
	"sync"

	eebusapi "github.com/enbility/eebus-go/api"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
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

// get returns the remote entity, nil if the use case has none (yet)
func (e *Entity[U]) get() spineapi.EntityRemoteInterface {
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

// Available returns the remote entity, ErrNotAvailable if the use case has none
// or the entity does not announce all of the given scenarios. Use case methods
// only validate the entity type, so the scenarios must be checked here.
func (e *Entity[U]) Available(scenarios ...uint) (spineapi.EntityRemoteInterface, error) {
	entity := e.get()
	if entity == nil {
		return nil, api.ErrNotAvailable
	}

	for _, scenario := range scenarios {
		if !e.uc.IsScenarioAvailableAtEntity(entity, scenario) {
			return nil, api.ErrNotAvailable
		}
	}

	return entity, nil
}

// Required is Available for a use case the device cannot operate without
func (e *Entity[U]) Required() (spineapi.EntityRemoteInterface, error) {
	entity := e.get()
	if entity == nil {
		return nil, ErrNotConnected
	}

	return entity, nil
}

// Read reads a use case value from the remote entity, reporting ErrNotAvailable
// while the scenario is unannounced or the value has not been received yet.
func (e *Entity[U]) Read[T any](read func(uc U, entity spineapi.EntityRemoteInterface) (T, error), scenarios ...uint) (T, error) {
	var zero T

	entity, err := e.Available(scenarios...)
	if err != nil {
		return zero, err
	}

	res, err := read(e.uc, entity)
	if err != nil {
		return zero, WrapError(err)
	}

	return res, nil
}

// Write runs a control write against the remote entity and waits for its result.
func (e *Entity[U]) Write(write func(uc U, entity spineapi.EntityRemoteInterface, cb ResultCB) (*model.MsgCounterType, error), scenarios ...uint) error {
	entity, err := e.Available(scenarios...)
	if err != nil {
		return err
	}

	return WrapError(Await(func(cb ResultCB) (*model.MsgCounterType, error) {
		return write(e.uc, entity, cb)
	}))
}

// WriteArg is Write for a control write taking an additional argument.
func (e *Entity[U]) WriteArg[A any](write func(uc U, entity spineapi.EntityRemoteInterface, arg A, cb ResultCB) (*model.MsgCounterType, error), arg A, scenarios ...uint) error {
	return e.Write(func(uc U, entity spineapi.EntityRemoteInterface, cb ResultCB) (*model.MsgCounterType, error) {
		return write(uc, entity, arg, cb)
	}, scenarios...)
}
