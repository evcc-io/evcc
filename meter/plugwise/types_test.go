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
