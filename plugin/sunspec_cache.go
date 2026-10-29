package plugin

import (
	"strconv"
	"sync"

	gosunspec "github.com/andig/gosunspec"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// sunspecDevice serializes access to a device tree shared by all values of one
// subdevice. gosunspec decodes into the shared model on every block read, hence
// a read and the point access following it must not interleave with another one.
type sunspecDevice struct {
	mu sync.Mutex
	*sunspec.SunSpec
}

var sunspecDevices = sunspecDeviceCache{
	data: make(map[string][]gosunspec.Device),
}

// sunspecDeviceCache is a cache for sunspec connection's device tree
type sunspecDeviceCache struct {
	data map[string][]gosunspec.Device
}

func (c *sunspecDeviceCache) Get(conn *modbus.Connection) []gosunspec.Device {
	return c.data[conn.Addr()]
}

func (c *sunspecDeviceCache) Put(conn *modbus.Connection, devices []gosunspec.Device) {
	c.data[conn.Addr()] = devices
}

var sunspecSubDevices = sunspecSubDeviceCache{
	data: make(map[string][]*sunspecDevice),
}

// sunspecSubDeviceCache is a cache for a sunspec devices's models
type sunspecSubDeviceCache struct {
	data map[string][]*sunspecDevice
}

func (c *sunspecSubDeviceCache) Get(conn *modbus.Connection, subDevice int) *sunspecDevice {
	addr := sunspecSubdeviceAddr(conn, subDevice)
	for _, dev := range c.data[addr] {
		if dev.Descriptor().SubDevice == subDevice {
			return dev
		}
	}

	return nil
}

func (c *sunspecSubDeviceCache) Put(conn *modbus.Connection, subDevice int, dev *sunspecDevice) {
	addr := sunspecSubdeviceAddr(conn, subDevice)
	c.data[addr] = append(c.data[addr], dev)
}

func sunspecSubdeviceAddr(conn *modbus.Connection, subDevice int) string {
	return conn.Addr() + "::" + strconv.Itoa(subDevice)
}
