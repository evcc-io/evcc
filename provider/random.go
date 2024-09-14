package provider

import (
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

type randomProvider struct {
	mu      sync.Mutex
	log     *util.Logger
	reset   *string
	set     Config
	timeout time.Duration
	cancel  func()
}

func init() {
	registry.Add("random", NewRandomFromConfig)
}

// NewRandomFromConfig creates random provider
func NewRandomFromConfig(other map[string]interface{}) (Provider, error) {
	var cc struct {
		Set Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	o := &randomProvider{
		log: util.NewLogger("random"),
		set: cc.Set,
	}

	return o, nil
}

var _ SetIntProvider = (*randomProvider)(nil)

func (o *randomProvider) IntSetter(param string) (func(int64) error, error) {
	set, err := NewIntSetterFromConfig(param, o.set)
	if err != nil {
		return nil, err
	}

	return func(int64) error {
		return set(rand.Int64N(math.MaxInt64-1) + 1)
	}, nil
}
