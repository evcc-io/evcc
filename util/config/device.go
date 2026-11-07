package config

import "sync"

type Device[T any] interface {
	Config() Named
	Instance() T
}

type ConfigurableDevice[T any] interface {
	Device[T]
	ID() int
	Properties() Properties
	Update(map[string]any, T, ...func(*Config)) error
	Delete() error
	SetRelease(func())
	Release()
}

var _ Device[any] = (*staticDevice[any])(nil)

type staticDevice[T any] struct {
	config   Named
	instance T
}

func NewStaticDevice[T any](config Named, instance T) Device[T] {
	return &staticDevice[T]{
		config:   config,
		instance: instance,
	}
}

func (d *staticDevice[T]) Config() Named {
	return d.config
}

func (d *staticDevice[T]) Instance() T {
	return d.instance
}

var _ ConfigurableDevice[any] = (*configurableDevice[any])(nil)

type configurableDevice[T any] struct {
	mu       sync.Mutex
	config   *Config
	instance T
	release  func()
}

func NewConfigurableDevice[T any](config *Config, instance T) ConfigurableDevice[T] {
	return &configurableDevice[T]{
		config:   config,
		instance: instance,
	}
}

func (d *configurableDevice[T]) Config() Named {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.config.Named()
}

func (d *configurableDevice[T]) Instance() T {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.instance
}

func (d *configurableDevice[T]) ID() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.config.ID
}

func (d *configurableDevice[T]) Properties() Properties {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.config.Properties
}

func (d *configurableDevice[T]) Update(config map[string]any, instance T, opt ...func(*Config)) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.config.Update(config, opt...); err != nil {
		return err
	}
	d.instance = instance
	return nil
}

func (d *configurableDevice[T]) Delete() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.config.Delete()
}

// SetRelease sets the function that releases the instance's resources, releasing the previous instance
func (d *configurableDevice[T]) SetRelease(release func()) {
	d.mu.Lock()
	prev := d.release
	d.release = release
	d.mu.Unlock()

	if prev != nil {
		prev()
	}
}

// Release releases the instance's resources
func (d *configurableDevice[T]) Release() {
	d.SetRelease(nil)
}
