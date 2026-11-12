package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNeighbors(t *testing.T) {
	tc := []struct {
		name, in string
		expected map[string]string
	}{
		{
			"linux",
			`IP address       HW type     Flags       HW address            Mask     Device
192.168.31.20    0x1         0x2         00:15:bb:12:34:56     *        eth0
192.168.31.99    0x1         0x0         00:00:00:00:00:00     *        eth0
`,
			map[string]string{"192.168.31.20": "00:15:BB:12:34:56"},
		},
		{
			"macos",
			`? (192.168.31.1) at 3c:37:12:a:b:c on en0 ifscope [ethernet]
? (192.168.31.7) at (incomplete) on en0 ifscope [ethernet]
? (192.168.31.255) at ff:ff:ff:ff:ff:ff on en0 ifscope [ethernet]
`,
			map[string]string{"192.168.31.1": "3C:37:12:0A:0B:0C"},
		},
		{
			"windows",
			`Interface: 192.168.31.5 --- 0x4
  Internet Address      Physical Address      Type
  192.168.31.21         00-03-ac-12-34-56     dynamic
  192.168.31.255        ff-ff-ff-ff-ff-ff     static
`,
			map[string]string{"192.168.31.21": "00:03:AC:12:34:56"},
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseNeighbors(tc.in))
		})
	}
}

func TestLocationHost(t *testing.T) {
	assert.Equal(t, "192.168.31.1", locationHost("http://192.168.31.1:49000/igddesc.xml"))
	assert.Empty(t, locationHost("://"))
}

func TestVendor(t *testing.T) {
	assert.Equal(t, "SMA Solar Technology AG", Vendor("00:15:BB:12:34:56"))
	// MA-M inside a shared MA-L block
	assert.Equal(t, "Victron Energy B.V.", Vendor("C0:61:9A:B1:23:45"))
	assert.Empty(t, Vendor("02:00:00:00:00:01"))
	assert.Empty(t, Vendor(""))
}
