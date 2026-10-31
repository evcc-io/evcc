package bluelink

import "testing"

func TestChargeControlURI(t *testing.T) {
	base := "https://au-apigw.ccs.hyundai.com.au:8080/api/v1/spa"

	got := chargeControlURI(base, "VID", true)
	want := "https://au-apigw.ccs.hyundai.com.au:8080/api/v2/spa/vehicles/VID/ccs2/control/charge"
	if got != want {
		t.Fatalf("ccs2 uri: got %s want %s", got, want)
	}

	got = chargeControlURI(base, "VID", false)
	want = "https://au-apigw.ccs.hyundai.com.au:8080/api/v1/spa/vehicles/VID/control/charge"
	if got != want {
		t.Fatalf("legacy uri: got %s want %s", got, want)
	}
}
