package eebus

import (
	"errors"
	"log"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/evcc-io/evcc/api"
)

func WrapError(err error) error {
	if errors.Is(err, eebusapi.ErrDataNotAvailable) {
		return api.ErrNotAvailable
	}
	return err
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
