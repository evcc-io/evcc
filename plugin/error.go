package plugin

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
)

type errorPlugin struct {
	err error
}

func init() {
	registry.Add("error", NewErrorFromConfig)
}

// NewErrorFromConfig creates error provider
func NewErrorFromConfig(other map[string]interface{}) (Plugin, error) {
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

	o := &errorPlugin{
		err: err,
	}

	return o, nil
}

var _ IntSetter = (*errorPlugin)(nil)

func (o *errorPlugin) IntSetter(param string) (func(int64) error, error) {
	return func(int64) error {
		return o.err
	}, nil
}
