package ble

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
)

func FindDevice(a *adapter.Adapter1, hwaddr string, timeout time.Duration) (*device.Device1, error) {
	dev, err := Discover(a, hwaddr, timeout)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, errors.New("device not found, is it advertising?")
	}

	return dev, nil
}

func Discover(a *adapter.Adapter1, hwaddr string, timeout time.Duration) (*device.Device1, error) {
	err := a.FlushDevices()
	if err != nil {
		return nil, err
	}

	discovery, cancel, err := api.Discover(a, nil)
	if err != nil {
		return nil, err
	}

	timer := time.NewTimer(timeout)

	for {
		select {
		case ev := <-discovery:
			dev, err := device.NewDevice1(ev.Path)
			if err != nil {
				return nil, err
			}
			if dev == nil || dev.Properties == nil {
				continue
			}

			p := dev.Properties
			if p.Address != hwaddr {
				continue
			}

			cancel()
			return dev, nil
		case <-timer.C:
			cancel()
			return nil, errors.New("discovery timeout exceeded")
		}
	}
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
