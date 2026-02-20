package meter

import (
	"sync"

	"github.com/evcc-io/evcc/api"
)

var _ api.Dimmer = (*Dimmer)(nil)

type Dimmer struct {
	mu     sync.Mutex
	active bool
	dimmed func() (bool, error)
	dim    func(bool) error
}

func NewDimmer(dim func(bool) error, dimmed func() (bool, error)) *Dimmer {
	return &Dimmer{
		dim:    dim,
		dimmed: dimmed,
	}
}

func (m *Dimmer) Dimmed() (bool, error) {
	if m.dimmed != nil {
		return m.dimmed()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.active, nil
}

func (m *Dimmer) Dim(enable bool) error {
	err := m.dim(enable)
	if err == nil {
		m.mu.Lock()
		m.active = enable
		m.mu.Unlock()
	}
	return err
}
