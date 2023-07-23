package config

type Device[T any] interface {
	Config() Named
	Connect(instance T)
	Instance() T
}
type DynamicDevice[T any] interface {
	Device[T]
	Update(Named, T)
}

type configurableDevice[T any] struct {
	config   Config
	instance T
}

func NewConfigurableDevice[T any](config Config) DynamicDevice[T] {
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

func (d *configurableDevice[T]) Update(Named, T) {
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
