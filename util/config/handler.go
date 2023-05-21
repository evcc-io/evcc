package config

import (
	"errors"
	"fmt"
	"sync"
)

type handler[T any] struct {
	mu        sync.Mutex
	container []container[T]
	visited   map[string]bool
}

// TrackVisitors tracks visited devices
func (cp *handler[T]) TrackVisitors() {
	cp.visited = make(map[string]bool)
}

// Add adds device and config
func (cp *handler[T]) Add(conf Named, device T) error {
	if conf.Name == "" {
		return errors.New("missing name")
	}

	if _, _, err := cp.ByName(conf.Name); err == nil {
		return fmt.Errorf("duplicate name: %s already defined and must be unique", conf.Name)
	}

	cp.container = append(cp.container, container[T]{device: device, config: conf})
	return nil
}

// Update updates device and config
func (cp *handler[T]) Update(conf Named, device T) error {
	if conf.Name == "" {
		return errors.New("missing name")
	}

	_, i, err := cp.ByName(conf.Name)
	if err != nil {
		return err
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.container[i] = container[T]{device: device, config: conf}

	return nil
}

// ByName provides device by name
func (cp *handler[T]) ByName(name string) (T, int, error) {
	var empty T

	cp.mu.Lock()
	defer cp.mu.Unlock()

	for i, container := range cp.container {
		if name == container.config.Name {
			// track duplicate usage https://github.com/evcc-io/evcc/issues/1744
			if cp.visited != nil {
				if _, ok := cp.visited[name]; ok {
					return empty, 0, fmt.Errorf("duplicate usage: %s", name)
				}
				cp.visited[name] = true
			}

			return container.device, i, nil
		}
	}

	return empty, 0, fmt.Errorf("does not exist: %s", name)
}

// Slice returns the slice of devices
func (cp *handler[T]) Slice() []T {
	res := make([]T, 0, len(cp.container))

	for _, container := range cp.container {
		res = append(res, container.device)
	}

	return res
}

// Map returns the map of devices
func (cp *handler[T]) Map() map[string]T {
	res := make(map[string]T, len(cp.container))

	for _, container := range cp.container {
		res[container.config.Name] = container.device
	}

	return res
}

// Config returns the configuration
func (cp *handler[T]) Config() []Named {
	res := make([]Named, 0, len(cp.container))

	for _, container := range cp.container {
		res = append(res, container.config)
	}

	return res
}
