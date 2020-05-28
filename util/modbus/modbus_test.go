package modbus

import "testing"

func TestParsePoint(t *testing.T) {
	tc := []struct {
		in           string
		model, block int
		point        string
		err          bool
	}{
		{"103:W", 103, 0, "W", false},
		{"802:1:V", 802, 1, "V", false},
		{"802::V", 802, 1, "V", true},
	}

	for _, tc := range tc {
		t.Log(tc)

		model, block, point, err := ParsePoint(tc.in)

		if (err != nil) != tc.err {
			t.Errorf("unexpected error: %d:%d:%s %v", model, block, point, err)
		}

		if !tc.err && (model != tc.model || block != tc.block || point != tc.point) {
			t.Errorf("unexpected result: %d:%d:%s", model, block, point)
		}
	}
}
