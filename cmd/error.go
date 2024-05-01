package cmd

type Class int

//go:generate enumer -type Class -trimprefix Class -transform=lower
const (
	_ Class = iota
	ClassMeter
	ClassCharger
	ClassVehicle
	ClassTariff
	ClassSite
	ClassMqtt
	ClassDatabase
	ClassEEBus
	ClassJavascript
	ClassGo
)

// DeviceError indicates the specific device that failed
type DeviceError struct {
	Name string
	err  error
}

func (e *DeviceError) Error() string {
	return e.err.Error()
}

// ClassError indicates the class of devices that failed
type ClassError struct {
	Class Class
	err   error
}

func (e *ClassError) Error() string {
	return e.err.Error()
}

func wrapErrorWithClass(class Class, err error) error {
	if err == nil {
		return nil
	}

	return &ClassError{
		Class: class,
		err:   err,
	}
}
