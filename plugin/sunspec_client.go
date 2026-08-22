package plugin

import (
	"bytes"
	"fmt"

	"github.com/evcc-io/evcc/util/modbus"
	gridx "github.com/grid-x/modbus"
)

// sunspecCachedClient dedups the holding register reads gosunspec issues on
// behalf of all sunspec values sharing a model block within one poll cycle.
type sunspecCachedClient struct {
	gridx.Client
	cache *modbus.Cache
}

// newSunspecCachedClient wraps client for use as gosunspec's modbus client.
func newSunspecCachedClient(client gridx.Client) *sunspecCachedClient {
	return &sunspecCachedClient{Client: client, cache: modbus.NewCache(modbusBlockTTL)}
}

func (c *sunspecCachedClient) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	key := fmt.Sprintf("%d/%d", address, quantity)

	b, _, err := c.cache.Fetch(key, func() ([]byte, error) {
		return c.Client.ReadHoldingRegisters(address, quantity)
	})
	if err != nil {
		return nil, err
	}

	// callers unmarshal in place, hand out a copy of the shared payload
	return bytes.Clone(b), nil
}

func (c *sunspecCachedClient) WriteSingleRegister(address, value uint16) ([]byte, error) {
	defer c.cache.Clear()
	return c.Client.WriteSingleRegister(address, value)
}

func (c *sunspecCachedClient) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	defer c.cache.Clear()
	return c.Client.WriteMultipleRegisters(address, quantity, value)
}
