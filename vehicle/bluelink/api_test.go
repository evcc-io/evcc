package bluelink

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestJoinErrors(t *testing.T) {
	ccs2Err := errors.New("ccs2 failed")
	legacyErr := errors.New("legacy failed")

	err := joinErrors(ccs2Err, legacyErr)
	if err == nil {
		t.Fatal("expected combined error")
	}
	if !errors.Is(err, legacyErr) {
		t.Fatalf("legacy error hidden: %v", err)
	}
	if !errors.Is(err, ccs2Err) {
		t.Fatalf("ccs2 error hidden: %v", err)
	}

	if err := joinErrors(ccs2Err); err != ccs2Err {
		t.Fatalf("single error wrapped: %v", err)
	}
}

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
	if err := decodeChargeControl([]byte("  \n")); err != nil {
		t.Fatalf("whitespace body: %v", err)
	}
	if err := decodeChargeControl([]byte(`{"retCode":"S"}`)); err != nil {
		t.Fatalf("success: %v", err)
	}
	if err := decodeChargeControl([]byte(`{}`)); err == nil {
		t.Fatal("expected error for missing retCode")
	}
	if err := decodeChargeControl([]byte(`{"msgId":"abc"}`)); err == nil {
		t.Fatal("expected error for body without retCode")
	}
	if err := decodeChargeControl([]byte(`null`)); err == nil {
		t.Fatal("expected error for null body")
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
	if res.ResMsg.ControlToken != "tok" {
		t.Fatalf("token: %s", res.ResMsg.ControlToken)
	}
	if res.ResMsg.ExpiresTime != 1700000000 {
		t.Fatalf("expires: %d", res.ResMsg.ExpiresTime)
	}

	var top pinResponse
	if err := json.Unmarshal([]byte(`{"controlToken":"legacy","expiresTime":1700000100}`), &top); err != nil {
		t.Fatal(err)
	}
	if top.ResMsg.ControlToken != "" || top.ControlToken != "legacy" {
		t.Fatalf("top-level fallback: %+v", top)
	}
}

func TestControlTokenExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	got := controlTokenExpiry(1_700_000_600, now)
	want := time.Unix(1_700_000_600, 0).Add(-30 * time.Second)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}

	past := controlTokenExpiry(1_699_999_000, now)
	if !past.Before(now) {
		t.Fatal("absolute expiresTime in the past must not keep the token cached")
	}
}

func TestSerialChargeLatestWins(t *testing.T) {
	s := newSerialCharge()
	started := make(chan struct{})
	release := make(chan struct{})

	var mu sync.Mutex
	var sent []bool
	var inflight atomic.Int32
	record := func(enable bool) error {
		n := inflight.Add(1)
		if n != 1 {
			t.Errorf("overlapping charge commands: %d", n)
		}
		mu.Lock()
		sent = append(sent, enable)
		mu.Unlock()
		inflight.Add(-1)
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		first := true
		_ = s.Do(true, func(enable bool) error {
			if first {
				first = false
				close(started)
				<-release
			}
			return record(enable)
		})
	}()
	<-started

	go func() {
		defer wg.Done()
		_ = s.Do(true, record)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		s.mu.Lock()
		pending := s.pending != nil && *s.pending
		s.mu.Unlock()
		if pending {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for queued start")
		}
		time.Sleep(time.Millisecond)
	}

	go func() {
		defer wg.Done()
		_ = s.Do(false, record)
	}()

	for {
		s.mu.Lock()
		pending := s.pending != nil && !*s.pending
		s.mu.Unlock()
		if pending {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for queued stop")
		}
		time.Sleep(time.Millisecond)
	}

	close(release)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(sent) != 2 || sent[0] != true || sent[1] != false {
		t.Fatalf("sent %v, want [true false]", sent)
	}
}
