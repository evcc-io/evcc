package meter

import (
	"sync"

	"github.com/evcc-io/evcc/api"
)

var _ api.Curtailer = (*Curtailer)(nil)

type Curtailer struct {
	mu        sync.Mutex
	active    bool
	curtailed func() (bool, error)
	curtail   func(bool) error
}

func NewCurtailer(curtail func(bool) error, curtailed func() (bool, error)) *Curtailer {
	return &Curtailer{
		curtail:   curtail,
		curtailed: curtailed,
	}
}

func (m *Curtailer) Curtailed() (bool, error) {
	if m.curtailed != nil {
		return m.curtailed()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.active, nil
}

func (m *Curtailer) Curtail(enable bool) error {
	err := m.curtail(enable)
	if err == nil {
		m.mu.Lock()
		m.active = enable
		m.mu.Unlock()
	}
	return err
}
