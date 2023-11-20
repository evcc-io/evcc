package provider

import (
	"github.com/evcc-io/evcc/util"
)

type sequenceProvider struct {
	set []Config
}

func init() {
	registry.Add("sequence", NewSequenceFromConfig)
}

// NewSequenceFromConfig creates sequence provider
func NewSequenceFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Set []Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &sequenceProvider{
		set: cc.Set,
	}

	return o, nil
}

var _ SetIntProvider = (*sequenceProvider)(nil)

func (o *sequenceProvider) IntSetter(param string) (func(int64) error, error) {
	set := make([]func(int64) error, 0, len(o.set))
	for _, cc := range o.set {
		s, err := NewIntSetterFromConfig(param, cc)
		if err != nil {
			return nil, err
		}
		set = append(set, s)
	}

	return func(val int64) error {
		for _, s := range set {
			if err := s(val); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

var _ SetFloatProvider = (*sequenceProvider)(nil)

func (o *sequenceProvider) FloatSetter(param string) (func(float64) error, error) {
	set := make([]func(float64) error, 0, len(o.set))
	for _, cc := range o.set {
		s, err := NewFloatSetterFromConfig(param, cc)
		if err != nil {
			return nil, err
		}
		set = append(set, s)
	}

	return func(val float64) error {
		for _, s := range set {
			if err := s(val); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

var _ SetBoolProvider = (*sequenceProvider)(nil)

func (o *sequenceProvider) BoolSetter(param string) (func(bool) error, error) {
	set := make([]func(bool) error, 0, len(o.set))
	for _, cc := range o.set {
		s, err := NewBoolSetterFromConfig(param, cc)
		if err != nil {
			return nil, err
		}
		set = append(set, s)
	}

	return func(val bool) error {
		for _, s := range set {
			if err := s(val); err != nil {
				return err
			}
		}
		return nil
	}, nil
}
