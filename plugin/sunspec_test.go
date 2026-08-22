package plugin

import (
	"sync"
	"testing"

	sunspec "github.com/andig/gosunspec"
	"github.com/andig/gosunspec/memory"
	"github.com/andig/gosunspec/models/model704"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sunsdev "github.com/volkszaehler/mbmd/meters/sunspec"
)

func TestSunspecBool(t *testing.T) {
	cases := []struct {
		res  int64
		mask uint64
		want bool
	}{
		{0, 0, false},
		{1, 0, true},
		{2, 0, true},
		{0b0100, 0b0010, false}, // masked-out bit
		{0b0110, 0b0010, true},  // masked-in bit
	}

	for _, c := range cases {
		if got := sunspecBool(c.res, c.mask); got != c.want {
			t.Errorf("sunspecBool(%v, %v) = %v, want %v", c.res, c.mask, got, c.want)
		}
	}
}

// newSunspecTestDevice builds an in-memory SunSpec model 704 (DER AC Controls)
// device; mbmd never touches the modbus.Client argument, so no connection is needed.
func newSunspecTestDevice(t *testing.T) (*sunspecDevice, sunspec.Block) {
	t.Helper()

	slab, err := memory.NewSlabBuilder().AddModel(model704.ModelID).Build()
	require.NoError(t, err)

	arr, err := memory.Open(slab)
	require.NoError(t, err)

	devices := arr.Collect(sunspec.AllDevices)
	require.NotEmpty(t, devices)

	block := devices[0].MustModel(sunspec.ModelId(model704.ModelID)).MustBlock(0)

	dev := &sunspecDevice{SunSpec: sunsdev.NewDevice("test")}
	require.NoError(t, dev.InitializeWithTree(devices))

	return dev, block
}

// TestSunspecBoolGetterEnum exercises BoolGetter against a real enum16 point
// (WMaxLimPctEna, SunSpec 704), the curtailment-enabled flag.
func TestSunspecBoolGetterEnum(t *testing.T) {
	dev, block := newSunspecTestDevice(t)

	for _, c := range []struct {
		val  sunspec.Enum16
		mask uint64
		want bool
	}{
		{0, 0, false},
		{1, 0, true},
		{0b11, 0b10, true},   // masked-in bit
		{0b11, 0b100, false}, // masked-out bit
	} {
		block.MustPoint(model704.WMaxLimPctEna).SetEnum16(c.val)
		require.NoError(t, block.Write(model704.WMaxLimPctEna))

		mb := &ModbusSunspec{
			device: dev,
			op:     modbus.SunSpecOperation{Model: model704.ModelID, Point: model704.WMaxLimPctEna},
			mask:   c.mask,
		}

		g, err := mb.BoolGetter()
		require.NoError(t, err)

		got, err := g()
		require.NoError(t, err)
		require.Equal(t, c.want, got, "value %v mask %v", c.val, c.mask)
	}
}

// TestSunspecBoolGetterInt exercises BoolGetter against a real scaled int-like
// point (WMaxLimPct, a uint16 with a SunSpec scale factor).
func TestSunspecBoolGetterInt(t *testing.T) {
	dev, block := newSunspecTestDevice(t)

	mb := &ModbusSunspec{
		device: dev,
		op:     modbus.SunSpecOperation{Model: model704.ModelID, Point: model704.WMaxLimPct},
	}

	g, err := mb.BoolGetter()
	require.NoError(t, err)

	block.MustPoint(model704.WMaxLimPct).SetUint16(0)
	require.NoError(t, block.Write(model704.WMaxLimPct))
	got, err := g()
	require.NoError(t, err)
	require.False(t, got)

	block.MustPoint(model704.WMaxLimPct).SetUint16(50)
	require.NoError(t, block.Write(model704.WMaxLimPct))
	got, err = g()
	require.NoError(t, err)
	require.True(t, got)
}

// TestSunspecConcurrentSharedDevice reads different points of one shared device
// tree concurrently, as the config page's parallel capability probes do.
func TestSunspecConcurrentSharedDevice(t *testing.T) {
	dev, block := newSunspecTestDevice(t)
	block.MustPoint(model704.WMaxLimPctEna).SetEnum16(1)
	block.MustPoint(model704.WMaxLimPct).SetUint16(50)
	require.NoError(t, block.Write(model704.WMaxLimPctEna))
	require.NoError(t, block.Write(model704.WMaxLimPct))

	var wg sync.WaitGroup
	for range 20 {
		for _, point := range []string{model704.WMaxLimPctEna, model704.WMaxLimPct} {
			wg.Go(func() {
				mb := &ModbusSunspec{
					device: dev,
					op:     modbus.SunSpecOperation{Model: model704.ModelID, Point: point},
				}

				g, err := mb.BoolGetter()
				assert.NoError(t, err) // require fails the test goroutine only

				_, err = g()
				assert.NoError(t, err)
			})
		}
	}
	wg.Wait()
}
