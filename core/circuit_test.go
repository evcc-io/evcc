package core

import "testing"

func TestCircuit(t *testing.T) {
	var lp1 interface{} = "lp1"
	var lp2 interface{} = "lp2"

	type testcase struct {
		lp      interface{}
		current float64
		enabled bool
		result  float64
	}

	tc := []testcase{
		{lp1, 0, false, 0},
		{lp2, 0, false, 0},
		{lp1, 20, false, 16},
		{lp2, 20, false, 16},
		{lp1, 20, true, 16},
		{lp2, 20, true, 0},
		{lp1, 8, true, 8},
		{lp2, 10, true, 8},
	}

	for _, tc := range tc {
		t.Logf("%v", tc)

		cur := circuit.Limit(tc.lp, tc.current, tc.enabled)
		if cur != tc.result {
			t.Errorf("Limit(%v, %v, %v) = %v, want %v", tc.lp, tc.current, tc.enabled, cur, tc.result)
		}

		// update circuit with last result
		circuit.Update(tc.lp, cur, tc.enabled)
	}
}
