package eebus

import (
	"errors"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/evcc-io/evcc/api"
)

func WrapError(err error) error {
	if errors.Is(err, eebusapi.ErrDataNotAvailable) {
		return api.ErrNotAvailable
	}
	return err
}
