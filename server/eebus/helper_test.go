package eebus

import (
	"errors"
	"testing"

	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestWrapError(t *testing.T) {
	// data not received yet is not an error condition
	for _, tc := range []error{eebusapi.ErrDataNotAvailable, eebusapi.ErrMetadataNotAvailable, eebusapi.ErrDataInvalid} {
		assert.ErrorIs(t, WrapError(tc), api.ErrNotAvailable, tc.Error())
	}

	assert.EqualError(t, WrapError(errors.New("boom")), "boom")
	assert.NoError(t, WrapError(nil))
}
