package eebus

import (
	"errors"
	"fmt"
	"log"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
)

// ErrNotConnected is returned while a remote entity required to operate the
// device is not (yet) available.
var ErrNotConnected = errors.New("not connected")

func WrapError(err error) error {
	if errors.Is(err, eebusapi.ErrDataNotAvailable) ||
		errors.Is(err, eebusapi.ErrMetadataNotAvailable) ||
		errors.Is(err, eebusapi.ErrDataInvalid) ||
		errors.Is(err, eebusapi.ErrNoCompatibleEntity) {
		return api.ErrNotAvailable
	}
	return err
}

// WriteTimeout bounds how long an awaited eebus write waits for its result.
const WriteTimeout = 10 * time.Second

// Await runs a control write and waits for the remote device's result, returning
// an error if the write is rejected or no result arrives within WriteTimeout.
// An unavailable use case is reported as ErrNotAvailable.
func Await(write func(func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error)) error {
	res := make(chan model.ResultDataType, 1)

	if _, err := write(func(r model.ResultDataType, _ model.MsgCounterType) { res <- r }); err != nil {
		return WrapError(err)
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
