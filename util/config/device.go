package config

type Device[T any] interface {
	Config() Named
	Connect(instance T)
	Instance() T
}
type ConfigurableDevice[T any] interface {
	Device[T]
	Update(map[string]any, T) error
}

type configurableDevice[T any] struct {
	config   Config
	instance T
}

func NewConfigurableDevice[T any](config Config) ConfigurableDevice[T] {
	return &configurableDevice[T]{config: config}
}

func (d *configurableDevice[T]) Config() Named {
	return d.config.Named()
}

func (d *configurableDevice[T]) Connect(instance T) {
	d.instance = instance
}

func (d *configurableDevice[T]) Instance() T {
	return d.instance
}

func (d *configurableDevice[T]) Update(config map[string]any, instance T) error {
	if err := d.config.Update(config); err != nil {
		return err
	}
	d.Connect(instance)
	return nil
}

type staticDevice[T any] struct {
	config   Named
	instance T
}

func NewStaticDevice[T any](config Named) Device[T] {
	return &staticDevice[T]{config: config}
}

func (d *staticDevice[T]) Configurable() bool {
	return true
}

func (d *staticDevice[T]) Config() Named {
	return d.config
}

func (d *staticDevice[T]) Connect(instance T) {
	d.instance = instance
}

func (d *staticDevice[T]) Instance() T {
	return d.instance
}
