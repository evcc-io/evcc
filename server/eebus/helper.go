package eebus

import (
	"errors"
	"fmt"
	"log"
	"time"

	eebusapi "github.com/enbility/eebus-go/api"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
)

func WrapError(err error) error {
	if errors.Is(err, eebusapi.ErrDataNotAvailable) {
		return api.ErrNotAvailable
	}
	return err
}

func wrapStartError(err error) error {
	if errors.Is(err, shipapi.ErrInvalidSKI) {
		const hint = "The stored EEBUS certificate has an invalid Subject Key Identifier (SKI).\n" +
			"The most common cause is a multi-year-old certificate whose SKI format is no longer accepted\n" +
			"by the stricter validation introduced in evcc 0.309.2 — see\n" +
			"https://github.com/evcc-io/evcc/issues/31366 for context.\n" +
			"To fix this, delete the EEBUS configuration and generate a new certificate:\n" +
			"  1. Open the evcc UI at Configuration > Services > EEBUS and remove the existing configuration, or\n" +
			"     delete the EEBUS section from your evcc.yaml.\n" +
			"  2. Generate a new certificate via the UI, or follow https://docs.evcc.io/de/reference/configuration/eebus/ for evcc.yaml.\n" +
			"  3. Re-pair each EEBUS device (wallbox, heat pump, etc.) with the new SKI.\n" +
			"§14a-EnWG users: do NOT run steps 1–3 on your production system yet. Stay on evcc < 0.309.2\n" +
			"with the old certificate; generate a new one on the side only to send its SKI to your\n" +
			"metering point operator for acceptance (this can take weeks). See\n" +
			"https://docs.evcc.io/de/features/external-control/ for the §14a feature."
		return fmt.Errorf("%w\n\n%s", err, hint)
	}
	return err
}

// WriteTimeout bounds how long an awaited eebus write waits for its result.
const WriteTimeout = 10 * time.Second

// Await runs a control write and waits for the remote device's result, returning
// an error if the write is rejected or no result arrives within WriteTimeout.
func Await(write func(func(model.ResultDataType)) (*model.MsgCounterType, error)) error {
	res := make(chan model.ResultDataType, 1)

	if _, err := write(func(r model.ResultDataType) { res <- r }); err != nil {
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
