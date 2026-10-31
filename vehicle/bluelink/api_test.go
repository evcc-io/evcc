package bluelink

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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

func TestDecodeChargeControl(t *testing.T) {
	if err := decodeChargeControl(nil); err != nil {
		t.Fatalf("empty body: %v", err)
	}
	if err := decodeChargeControl([]byte(`{"retCode":"S"}`)); err != nil {
		t.Fatalf("success: %v", err)
	}
	if err := decodeChargeControl([]byte(`{}`)); err == nil {
		t.Fatal("expected error for missing retCode")
	}
	if err := decodeChargeControl([]byte(`{"retCode":"F"}`)); err == nil {
		t.Fatal("expected error for failed retCode")
	}
}

func TestPinResponseResMsg(t *testing.T) {
	var res pinResponse
	if err := json.Unmarshal([]byte(`{"retCode":"S","resMsg":{"controlToken":"tok","expiresTime":1700000000}}`), &res); err != nil {
		t.Fatal(err)
	}
	if res.token() != "tok" {
		t.Fatalf("token: %s", res.token())
	}
	if res.expiresTime() != 1700000000 {
		t.Fatalf("expires: %d", res.expiresTime())
	}
}

func TestControlTokenExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	got := controlTokenExpiry(1_700_000_600, now)
	want := time.Unix(1_700_000_600, 0).Add(-30 * time.Second)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestLockVehicleSerializes(t *testing.T) {
	api := &API{chargeMu: make(map[string]*sync.Mutex)}
	var current, max atomic.Int32
	var wg sync.WaitGroup

	run := func() {
		defer wg.Done()
		mu := api.lockVehicle("VID")
		mu.Lock()
		defer mu.Unlock()
		n := current.Add(1)
		for {
			old := max.Load()
			if n <= old || max.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		current.Add(-1)
	}

	wg.Add(2)
	go run()
	go run()
	wg.Wait()

	if max.Load() != 1 {
		t.Fatalf("overlapping charge commands: %d", max.Load())
	}
}
