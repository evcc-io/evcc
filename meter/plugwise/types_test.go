package plugwise

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalDomainObjects(t *testing.T) {
	data, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err)

	var res DomainObjects
	require.NoError(t, xml.Unmarshal(data, &res))

	// electricity_consumed point_log (unit=W): nl_offpeak=0.00 + nl_peak=312.00 = 312.0
	consumed := res.Location.Logs.PowerWatts("electricity_consumed")
	assert.Equal(t, 312.0, consumed)

	// electricity_produced point_log (unit=W): nl_offpeak=0.00 + nl_peak=0.00 = 0.0
	produced := res.Location.Logs.PowerWatts("electricity_produced")
	assert.Equal(t, 0.0, produced)

	// Net power: consumed - produced = 312.0 W (positive = grid import)
	assert.Equal(t, 312.0, consumed-produced)
}

func TestVoltageVolts(t *testing.T) {
	data, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err)

	var res DomainObjects
	require.NoError(t, xml.Unmarshal(data, &res))

	// Valid phases — voltage point_logs with unit=V, single measurement, no tariff attribute.
	assert.Equal(t, 232.3, res.Location.Logs.VoltageVolts("voltage_phase_one"))
	assert.Equal(t, 229.8, res.Location.Logs.VoltageVolts("voltage_phase_two"))
	assert.Equal(t, 231.2, res.Location.Logs.VoltageVolts("voltage_phase_three"))

	// Absent type — loop never matches → 0.0.
	assert.Equal(t, 0.0, res.Location.Logs.VoltageVolts("does_not_exist"))

	// Unit-filter correctness: electricity_consumed exists but has Unit=="W", NOT "V".
	// The V-filter must exclude it and return 0.0.
	assert.Equal(t, 0.0, res.Location.Logs.VoltageVolts("electricity_consumed"))
}
