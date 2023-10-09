package config

type Device[T any] interface {
	Config() Named
	Instance() T
}
type ConfigurableDevice[T any] interface {
	Device[T]
	ID() int
	Update(map[string]any, T) error
	Delete() error
}

type configurableDevice[T any] struct {
	config   Config
	instance T
}

func NewConfigurableDevice[T any](config Config, instance T) ConfigurableDevice[T] {
	return &configurableDevice[T]{
		config:   config,
		instance: instance,
	}
}

func (d *configurableDevice[T]) Config() Named {
	return d.config.Named()
}

func (d *configurableDevice[T]) Instance() T {
	return d.instance
}

func (d *configurableDevice[T]) ID() int {
	return d.config.ID
}

func (d *configurableDevice[T]) Update(config map[string]any, instance T) error {
	if err := d.config.Update(config); err != nil {
		return err
	}
	d.instance = instance
	return nil
}

func (d *configurableDevice[T]) Delete() error {
	return d.config.Delete()
}

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

func (d *staticDevice[T]) Configurable() bool {
	return true
}

func (d *staticDevice[T]) Config() Named {
	return d.config
}

func (d *staticDevice[T]) Instance() T {
	return d.instance
}
