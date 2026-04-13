package danfoss

import (
	"errors"
	"fmt"
)

// Discover sends a broadcast PingRequest and returns the address of the
// inverter on the bus. If exactly one inverter replies it is returned. If
// multiple inverters reply an error is returned instructing the caller to set
// the node address explicitly in the configuration (one meter entry per
// inverter).
func Discover(c *Client) (Address, error) {
	addrs, err := c.Ping(true)
	if err != nil {
		return Address{}, fmt.Errorf("ping: %w", err)
	}
	switch len(addrs) {
	case 0:
		return Address{}, errors.New("no ComLynx inverter found on the bus; check RS485 wiring")
	case 1:
		return addrs[0], nil
	default:
		list := make([]string, len(addrs))
		for i, a := range addrs {
			list[i] = a.String()
		}
		return Address{}, fmt.Errorf(
			"%d inverters found on bus (%v): set 'node' in config to target one inverter explicitly",
			len(addrs), list)
	}
}
