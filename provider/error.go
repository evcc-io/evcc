package provider

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type errorProvider struct {
	err error
}

func init() {
	registry.Add("error", NewErrorFromConfig)
}

// NewErrorFromConfig creates error provider
func NewErrorFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Error string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	err := knownErrors([]byte(cc.Error))
	if err == nil {
		return nil, fmt.Errorf("unknown error: %s", cc.Error)
	}

	o := &errorProvider{
		err: err,
	}

	return o, nil
}

var _ SetIntProvider = (*errorProvider)(nil)

func (o *errorProvider) IntSetter(param string) (func(int64) error, error) {
	return func(int64) error {
		return o.err
	}, nil
}
