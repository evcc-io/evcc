package nrgble

import (
	"errors"
	"fmt"
	"strings"

	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
)

func FindDevice(a *adapter.Adapter1, hwaddr string) (*device.Device1, error) {
	dev, err := Discover(a, hwaddr)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, errors.New("Device not found, is it advertising?")
	}

	return dev, nil
}

func Discover(a *adapter.Adapter1, hwaddr string) (*device.Device1, error) {
	err := a.FlushDevices()
	if err != nil {
		return nil, err
	}

	discovery, cancel, err := api.Discover(a, nil)
	if err != nil {
		return nil, err
	}

	defer cancel()

	for ev := range discovery {
		dev, err1 := device.NewDevice1(ev.Path)
		if err != nil {
			return nil, err1
		}

		if dev == nil || dev.Properties == nil {
			continue
		}

		p := dev.Properties

		if p.Address != hwaddr {
			continue
		}

		return dev, nil
	}

	return nil, nil
}

func Connect(dev *device.Device1, ag *agent.SimpleAgent, adapterID string) error {
	props, err := dev.GetProperties()
	if err != nil {
		return fmt.Errorf("Failed to load props: %s", err)
	}

	if props.Connected {
		return nil
	}

	if !props.Paired || !props.Trusted {
		if err := dev.Pair(); err != nil {
			return fmt.Errorf("Pair failed: %s", err)
		}

		if err := agent.SetTrusted(adapterID, dev.Path()); err != nil {
			return fmt.Errorf("Set trusted failed: %s", err)
		}
	}

	if !props.Connected {
		err = dev.Connect()
		if err != nil {
			if !strings.Contains(err.Error(), "Connection refused") {
				return fmt.Errorf("Connect failed: %s", err)
			}
		}
	}

	return nil
}
