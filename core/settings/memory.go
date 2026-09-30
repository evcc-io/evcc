package settings

import (
	"encoding/json"
	"errors"
	"sync"
)

var _ Settings = (*memorySettings)(nil)

type memorySettings struct {
	accessor
	mu   sync.Mutex
	vals map[string]any
}

// NewMemorySettings creates a non-persistent settings store
func NewMemorySettings() Settings {
	s := &memorySettings{vals: make(map[string]any)}
	s.accessor = accessor{s.get, s.set}
	return s
}

func (s *memorySettings) get(key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.vals[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return val, nil
}

func (s *memorySettings) set(key string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vals[key] = val
}

func (s *memorySettings) SetJson(key string, val any) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	s.set(key, string(b))
	return nil
}

func (s *memorySettings) Json(key string, res any) error {
	str, err := s.String(key)
	if str == "" || err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), &res)
}
