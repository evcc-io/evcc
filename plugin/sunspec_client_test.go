package plugin

import (
	"errors"
	"testing"

	"github.com/andig/gosunspec/memory"
	"github.com/andig/gosunspec/models/model704"
	gridx "github.com/grid-x/modbus"
	"github.com/stretchr/testify/require"
	sunsdev "github.com/volkszaehler/mbmd/meters/sunspec"
)

// countingClient counts the physical reads reaching the device, failing the
// first failReads of them
type countingClient struct {
	gridx.Client
	reads     int
	failReads int
}

func (c *countingClient) ReadHoldingRegisters(_, quantity uint16) ([]byte, error) {
	c.reads++
	if c.reads <= c.failReads {
		return nil, errors.New("read failed")
	}
	return make([]byte, 2*quantity), nil
}

func (c *countingClient) WriteSingleRegister(_, _ uint16) ([]byte, error) {
	return nil, nil
}

func TestSunspecCachedClient(t *testing.T) {
	cnt := new(countingClient)
	c := newSunspecCachedClient(cnt)

	for range 3 {
		_, err := c.ReadHoldingRegisters(40000, 4)
		require.NoError(t, err)
	}
	require.Equal(t, 1, cnt.reads, "reads of the same block must share one exchange")

	_, err := c.ReadHoldingRegisters(40100, 4)
	require.NoError(t, err)
	require.Equal(t, 2, cnt.reads, "different block must be read separately")

	// a write invalidates the cache
	_, err = c.WriteSingleRegister(40000, 1)
	require.NoError(t, err)

	_, err = c.ReadHoldingRegisters(40000, 4)
	require.NoError(t, err)
	require.Equal(t, 3, cnt.reads, "read after write must not serve stale data")
}

func TestSunspecCachedClientError(t *testing.T) {
	cnt := &countingClient{failReads: 1}
	c := newSunspecCachedClient(cnt)

	_, err := c.ReadHoldingRegisters(40000, 4)
	require.Error(t, err, "read error must be propagated")

	// errors are not cached, the next read reaches the device again
	_, err = c.ReadHoldingRegisters(40000, 4)
	require.NoError(t, err)
	require.Equal(t, 2, cnt.reads)

	_, err = c.ReadHoldingRegisters(40000, 4)
	require.NoError(t, err)
	require.Equal(t, 2, cnt.reads, "successful read must be cached afterwards")
}

func TestSunspecCachedClientPayloadIsolated(t *testing.T) {
	c := newSunspecCachedClient(new(countingClient))

	b, err := c.ReadHoldingRegisters(40000, 1)
	require.NoError(t, err)
	b[0] = 0xff

	b, err = c.ReadHoldingRegisters(40000, 1)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0}, b, "cached payload must not be mutated by callers")
}

const slabBase = 40000

// slabClient serves a gosunspec memory slab as a modbus device, counting the
// register reads a real device would have to answer.
type slabClient struct {
	gridx.Client
	slab  []byte
	reads int
}

func (s *slabClient) offset(address, quantity uint16) (int, error) {
	if address < slabBase || int(address-slabBase)*2+int(quantity)*2 > len(s.slab) {
		return 0, errors.New("bad address")
	}
	return int(address-slabBase) * 2, nil
}

func (s *slabClient) ReadHoldingRegisters(address, quantity uint16) ([]byte, error) {
	off, err := s.offset(address, quantity)
	if err != nil {
		return nil, err
	}
	s.reads++
	return s.slab[off : off+int(quantity)*2], nil
}

func (s *slabClient) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	off, err := s.offset(address, quantity)
	if err != nil {
		return nil, err
	}
	copy(s.slab[off:off+int(quantity)*2], value)
	return nil, nil
}

func (s *slabClient) WriteSingleRegister(address, value uint16) ([]byte, error) {
	return s.WriteMultipleRegisters(address, 1, []byte{byte(value >> 8), byte(value)})
}

// TestSunspecCachedClientSharedBlock exercises the full gosunspec read path:
// values pointing at different points of the same model block must collapse
// into a single physical block read.
func TestSunspecCachedClientSharedBlock(t *testing.T) {
	slab, err := memory.NewSlabBuilder().AddModel(model704.ModelID).Build()
	require.NoError(t, err)

	sim := &slabClient{slab: slab}
	c := newSunspecCachedClient(sim)

	devices, err := sunsdev.DeviceTree(c)
	require.NoError(t, err)

	dev := sunsdev.NewDevice("test")
	require.NoError(t, dev.InitializeWithTree(devices))

	c.cache.Clear()
	sim.reads = 0

	_, _, err = dev.QueryPointAny(c, model704.ModelID, 0, model704.WMaxLimPct)
	require.NoError(t, err)
	require.NotZero(t, sim.reads)

	reads := sim.reads
	_, _, err = dev.QueryPointAny(c, model704.ModelID, 0, model704.WMaxLimPctEna)
	require.NoError(t, err)
	require.Equal(t, reads, sim.reads, "second point of the same block must be served from cache")

	// a write must invalidate the cached block
	block, point, err := dev.QueryPointAny(c, model704.ModelID, 0, model704.WMaxLimPct)
	require.NoError(t, err)
	point.SetUint16(42)
	require.NoError(t, block.Write(model704.WMaxLimPct))

	_, _, err = dev.QueryPointAny(c, model704.ModelID, 0, model704.WMaxLimPctEna)
	require.NoError(t, err)
	require.Greater(t, sim.reads, reads, "read after write must not serve stale data")
}
