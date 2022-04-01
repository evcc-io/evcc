package idkproxy

import "testing"

func TestQmAuth(t *testing.T) {
	res := qmauth(16473775)
	exp := "d1d93982777c56ed2f0d07c4b66435e821fd69a771c3777e43c3cdc219819107"

	if res != exp {
		t.Errorf("expected %s, got %s", exp, res)
	}
}
