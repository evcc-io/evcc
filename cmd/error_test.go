package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	res := &ClassError{
		ClassMeter,
		&DeviceError{
			"0815",
			errors.New("foo"),
		},
	}

	b, err := res.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, `{"class":"meter","device":"0815","error":"foo"}`, string(b))
}
