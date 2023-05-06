package homematic

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test MethodResponse response
func TestUnmarshalMethodResponse(t *testing.T) {

	{
		// BidCos-RF (Port 2001) getParamset measure-channel response test
		var res MethodResponse

		xmlstr := `<?xml version="1.0" encoding="iso-8859-1"?><methodResponse><params><param><value><struct><member><name>IEC_ENERGY_COUNTER</name><value><double>689.586500</double></value></member><member><name>IEC_POWER</name><value><double>166.390000</double></value></member></struct></value></param></params></methodResponse>`
		assert.NoError(t, xml.Unmarshal([]byte(strings.Replace(string(xmlstr), "iso-8859-1", "UTF-8", 1)), &res))

		assert.Equal(t, "IEC_ENERGY_COUNTER", res.Member[0].Name)
		assert.Equal(t, float64(689.586500), res.Member[0].Value.CCUFloat)
	}

	{
		// BidCos-IP (Port 2010) getParamset measure-channel response test
		var res MethodResponse

		xmlstr := `<?xml version="1.0" encoding="ISO-8859-1"?><methodResponse><params><param><value><struct><member><name>VOLTAGE</name><value><double>230.6</double></value></member><member><name>POWER_STATUS</name><value><i4>0</i4></value></member><member><name>ENERGY_COUNTER</name><value><double>10888.7</double></value></member><member><name>CURRENT_STATUS</name><value><i4>0</i4></value></member><member><name>FREQUENCY</name><value><double>49.97</double></value></member><member><name>ENERGY_COUNTER_OVERFLOW</name><value><boolean>0</boolean></value></member><member><name>POWER</name><value><double>0.05</double></value></member><member><name>VOLTAGE_STATUS</name><value><i4>0</i4></value></member><member><name>CURRENT</name><value><double>0.0</double></value></member><member><name>FREQUENCY_STATUS</name><value><i4>0</i4></value></member></struct></value></param></params></methodResponse>`
		assert.NoError(t, xml.Unmarshal([]byte(strings.Replace(string(xmlstr), "ISO-8859-1", "UTF-8", 1)), &res))

		assert.Equal(t, "ENERGY_COUNTER", res.Member[2].Name)
		assert.Equal(t, float64(10888.7), res.Member[2].Value.CCUFloat)
		assert.Equal(t, "POWER", res.Member[6].Name)
		assert.Equal(t, float64(0.05), res.Member[6].Value.CCUFloat)
	}

	{
		// BidCos-IP (Port 2010) getParamset switch-channel response test
		var res MethodResponse

		xmlstr := `<?xml version="1.0" encoding="ISO-8859-1"?><methodResponse><params><param><value><struct><member><name>SECTION_STATUS</name><value><i4>0</i4></value></member><member><name>PROCESS</name><value><i4>0</i4></value></member><member><name>STATE</name><value><boolean>1</boolean></value></member><member><name>SECTION</name><value><i4>2</i4></value></member></struct></value></param></params></methodResponse>`
		assert.NoError(t, xml.Unmarshal([]byte(strings.Replace(string(xmlstr), "ISO-8859-1", "UTF-8", 1)), &res))

		assert.Equal(t, "STATE", res.Member[2].Name)
		assert.Equal(t, true, res.Member[2].Value.CCUBool)
	}

	{
		// Faulty response test
		var res MethodResponse

		xmlstr := `<?xml version="1.0" encoding="ISO-8859-1"?><methodResponse><fault><value><struct><member><name>faultCode</name><value><i4>-2</i4></value></member><member><name>faultString</name><value>Invalid device</value></member></struct></value></fault></methodResponse>`
		assert.NoError(t, xml.Unmarshal([]byte(strings.Replace(string(xmlstr), "ISO-8859-1", "UTF-8", 1)), &res))

		assert.Equal(t, "faultCode", res.Fault[0].Name)
		assert.Equal(t, int64(-2), res.Fault[0].Value.CCUInt)
		assert.Equal(t, "faultString", res.Fault[1].Name)
		assert.Equal(t, "Invalid device", res.Fault[1].Value.CCUString)
	}

}
