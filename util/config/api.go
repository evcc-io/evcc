package config

// DeviceProperties returns the common device data for the given reference
func DeviceProperties[T any](dev Device[T]) Properties {
	if d, ok := dev.(ConfigurableDevice[T]); ok {
		return d.Properties()
	}
	return Properties{}
}

// DeviceTitleOrName returns device title or name
func DeviceTitleOrName[T any](dev Device[T]) string {
	if d, ok := dev.(ConfigurableDevice[T]); ok {
		if title := d.Properties().Title; title != "" {
			return title
		}
	}
	return dev.Config().Name
}
