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
		// Double response test
		var res MethodResponse

		xmlstr := `<?xml version="1.0" encoding="ISO-8859-1"?><methodResponse><params><param><value><double>20698.0</double></value></param></params></methodResponse>`
		assert.NoError(t, xml.Unmarshal([]byte(strings.Replace(string(xmlstr), "ISO-8859-1", "UTF-8", 1)), &res))

		assert.Equal(t, float64(20698), res.Value.CCUFloat)
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
