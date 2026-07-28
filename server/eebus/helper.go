package eebus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	eebusapi "github.com/enbility/eebus-go/api"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// ErrNotConnected is returned while a remote entity required to operate the
// device is not (yet) available.
var ErrNotConnected = errors.New("not connected")

func WrapError(err error) error {
	if errors.Is(err, eebusapi.ErrDataNotAvailable) ||
		errors.Is(err, eebusapi.ErrMetadataNotAvailable) ||
		errors.Is(err, eebusapi.ErrDataInvalid) {
		return api.ErrNotAvailable
	}
	return err
}

// RequiredEntity returns the remote entity while the use case scenario is
// available at it and ErrNotConnected otherwise. Use for entities the device
// cannot operate without; optional data uses ReadValue.
func RequiredEntity(uc eebusapi.UseCaseBaseInterface, entity spineapi.EntityRemoteInterface, scenario uint) (spineapi.EntityRemoteInterface, error) {
	if entity == nil || !uc.IsScenarioAvailableAtEntity(entity, scenario) {
		return nil, ErrNotConnected
	}

	return entity, nil
}

// ReadValue reads a use case value from the remote entity. It reports
// ErrNotAvailable while the scenario is unavailable at the entity or the value
// has not been received yet.
func ReadValue[T any](uc eebusapi.UseCaseBaseInterface, entity spineapi.EntityRemoteInterface, scenario uint, read func(entity spineapi.EntityRemoteInterface) (T, error)) (T, error) {
	var zero T

	if entity == nil || !uc.IsScenarioAvailableAtEntity(entity, scenario) {
		return zero, api.ErrNotAvailable
	}

	res, err := read(entity)
	if err != nil {
		return zero, WrapError(err)
	}

	return res, nil
}

// UpdateEntity returns the remote entity to cache for a use case after a use case
// support update. Removing a use case from an entity emits the same event with the
// entity still set, so the cached entity is dropped once its scenarios are gone.
// Of the remaining candidates the least specific (shallowest) entity wins.
func UpdateEntity(uc eebusapi.UseCaseBaseInterface, cached, entity spineapi.EntityRemoteInterface) spineapi.EntityRemoteInterface {
	if cached != nil && len(uc.AvailableScenariosForEntity(cached)) == 0 {
		cached = nil
	}

	// removal, or an entity that doesn't support any scenario of the use case
	if entity == nil || len(uc.AvailableScenariosForEntity(entity)) == 0 {
		return cached
	}

	if cached == nil || len(entity.Address().Entity) < len(cached.Address().Entity) {
		return entity
	}

	return cached
}

// WriteTimeout bounds how long an awaited eebus write waits for its result.
const WriteTimeout = 10 * time.Second

// Await runs a control write and waits for the remote device's result, returning
// an error if the write is rejected or no result arrives within WriteTimeout.
func Await(write func(func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error)) error {
	res := make(chan model.ResultDataType, 1)

	if _, err := write(func(r model.ResultDataType, _ model.MsgCounterType) { res <- r }); err != nil {
		return err
	}

	select {
	case r := <-res:
		if r.ErrorNumber != nil && *r.ErrorNumber != 0 {
			err := fmt.Errorf("write rejected: %d", *r.ErrorNumber)
			if r.Description != nil {
				err = fmt.Errorf("%w (%s)", err, *r.Description)
			}
			return err
		}
		return nil
	case <-time.After(WriteTimeout):
		return errors.New("write result timeout")
	}
}

// limitTimeout keeps the retries inside the 60s the spec grants the Energy Guard.
const limitTimeout = 50 * time.Second

// AssertLimit states the current limit to a newly available Controllable System
// ([LPC-913]/[LPP-913]). Blocks while retrying- the CS ignores writes that do not
// follow a heartbeat and may reject them while still in state "init". Retrying
// stops when ctx is cancelled, i.e. when the device is gone.
func AssertLimit(ctx context.Context, log *util.Logger, write func() error) {
	bo := backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(limitTimeout))
	if err := backoff.Retry(write, backoff.WithContext(bo, ctx)); err != nil && ctx.Err() == nil {
		log.DEBUG.Printf("assert limit: %v", err)
	}
}

func LogEntities(log *log.Logger, actor string, uc eebusapi.UseCaseInterface) {
	ss := uc.RemoteEntitiesScenarios()
	if len(ss) > 0 {
		log.Printf("%s:", actor)
	}

	for _, s := range ss {
		var desc string
		if d := s.Entity.Description(); d != nil {
			desc = string(*d)
		}

		log.Printf("  entity: %s scenarios: %v meta: %s (%s)", s.Entity.Address(), s.Scenarios, s.Entity.EntityType(), desc)
	}
}
