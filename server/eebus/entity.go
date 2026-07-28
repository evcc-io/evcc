package eebus

import (
	"sync"

	eebusapi "github.com/enbility/eebus-go/api"
	spineapi "github.com/enbility/spine-go/api"
)

// Entity is the remote entity a use case is available at. It guards its own
// access, the zero value is ready to use.
type Entity struct {
	mu     sync.RWMutex
	entity spineapi.EntityRemoteInterface
}

// Get returns the remote entity, nil if the use case has none (yet)
func (e *Entity) Get() spineapi.EntityRemoteInterface {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.entity
}

// Set records the remote entity, use nil to drop it e.g. on disconnect
func (e *Entity) Set(entity spineapi.EntityRemoteInterface) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.entity = entity
}

// Update records the remote entity reported by a use case support update. Removal
// emits the same event with the entity set, so a recorded entity is dropped once
// its scenarios are gone; otherwise the shallowest entity wins.
func (e *Entity) Update(uc eebusapi.UseCaseBaseInterface, entity spineapi.EntityRemoteInterface) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.entity != nil && len(uc.AvailableScenariosForEntity(e.entity)) == 0 {
		e.entity = nil
	}

	// removal, or an entity that doesn't support any scenario of the use case
	if entity == nil || len(uc.AvailableScenariosForEntity(entity)) == 0 {
		return
	}

	if e.entity == nil || len(entity.Address().Entity) < len(e.entity.Address().Entity) {
		e.entity = entity
	}
}

// Required returns the remote entity while the use case scenario is available at
// it, ErrNotConnected otherwise. Optional data uses ReadValue.
func (e *Entity) Required(uc eebusapi.UseCaseBaseInterface, scenario uint) (spineapi.EntityRemoteInterface, error) {
	entity := e.Get()

	if entity == nil || !uc.IsScenarioAvailableAtEntity(entity, scenario) {
		return nil, ErrNotConnected
	}

	return entity, nil
}
