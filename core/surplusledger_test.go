package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
)

// TestSurplusLedger covers claim accumulation and time-based reconciliation.
func TestSurplusLedger(t *testing.T) {
	clk := clock.NewMock()
	l := newSurplusLedger(clk, time.Minute)

	if got := l.inflight(); got != 0 {
		t.Fatalf("empty ledger inflight = %v, want 0", got)
	}

	l.claim(1000)
	if got := l.inflight(); got != 1000 {
		t.Fatalf("after claim inflight = %v, want 1000", got)
	}

	clk.Add(30 * time.Second)
	l.claim(-400) // a decrease frees budget back
	if got := l.inflight(); got != 600 {
		t.Fatalf("after decrease inflight = %v, want 600", got)
	}

	// the first claim (now 61s old) expires once the meters reflect it; the
	// second (31s old) is still in flight
	clk.Add(31 * time.Second)
	if got := l.inflight(); got != -400 {
		t.Fatalf("after first expiry inflight = %v, want -400", got)
	}

	// the second claim expires too
	clk.Add(30 * time.Second)
	if got := l.inflight(); got != 0 {
		t.Fatalf("after full expiry inflight = %v, want 0", got)
	}
}

// TestSurplusLedgerDisabled verifies a non-positive settle disables the ledger.
func TestSurplusLedgerDisabled(t *testing.T) {
	l := newSurplusLedger(clock.NewMock(), 0)
	l.claim(1000)
	if got := l.inflight(); got != 0 {
		t.Fatalf("disabled ledger inflight = %v, want 0", got)
	}
}

func newLedgerLoadpoint(clk clock.Clock, charger api.Charger, ledger *surplusLedger) *Loadpoint {
	return &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clk,
		bus:         evbus.New(),
		charger:     charger,
		wakeUpTimer: NewTimer(),
		minCurrent:  minA,
		maxCurrent:  maxA,
		phases:      1,
		ledger:      ledger,
	}
}

// TestSetLimitRecordsLedgerClaim verifies that an actuation records its
// un-metered power delta against the ledger.
func TestSetLimitRecordsLedgerClaim(t *testing.T) {
	Voltage = 230

	clk := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	ledger := newSurplusLedger(clk, time.Minute)
	lp := newLedgerLoadpoint(clk, charger, ledger)

	// enable from disabled at min current -> claim = currentToPower(minA, 1p)
	charger.EXPECT().MaxCurrent(int64(minA)).Return(nil)
	charger.EXPECT().Enable(true).Return(nil)
	if err := lp.setLimit(minA); err != nil {
		t.Fatalf("setLimit: %v", err)
	}

	if got, want := ledger.inflight(), currentToPower(minA, 1); got != want {
		t.Fatalf("inflight after enable = %v, want %v", got, want)
	}

	// disabling frees the whole claim back (net zero)
	charger.EXPECT().Enable(false).Return(nil)
	if err := lp.setLimit(0); err != nil {
		t.Fatalf("setLimit(0): %v", err)
	}
	if got := ledger.inflight(); got != 0 {
		t.Fatalf("inflight after disable = %v, want 0", got)
	}
}

// TestSurplusLedgerTwoLoadpoints verifies that successive actuations on two
// loadpoints sharing a ledger accumulate, so the site discounts the combined
// in-flight claims — the second loadpoint cannot re-grab surplus the first
// already took. site.update passes sitePower + ledger.inflight() to control(),
// so an inflight of 2×draw shifts the effective surplus by that amount.
func TestSurplusLedgerTwoLoadpoints(t *testing.T) {
	Voltage = 230

	clk := clock.NewMock()
	ctrl := gomock.NewController(t)
	ledger := newSurplusLedger(clk, time.Minute)

	charger1 := api.NewMockCharger(ctrl)
	charger2 := api.NewMockCharger(ctrl)
	lp1 := newLedgerLoadpoint(clk, charger1, ledger)
	lp2 := newLedgerLoadpoint(clk, charger2, ledger)

	draw := currentToPower(minA, 1)

	// first loadpoint enables
	charger1.EXPECT().MaxCurrent(int64(minA)).Return(nil)
	charger1.EXPECT().Enable(true).Return(nil)
	if err := lp1.setLimit(minA); err != nil {
		t.Fatalf("lp1 setLimit: %v", err)
	}
	if got := ledger.inflight(); got != draw {
		t.Fatalf("inflight after lp1 = %v, want %v", got, draw)
	}

	// second loadpoint enables shortly after, before the meters caught up
	clk.Add(time.Second)
	charger2.EXPECT().MaxCurrent(int64(minA)).Return(nil)
	charger2.EXPECT().Enable(true).Return(nil)
	if err := lp2.setLimit(minA); err != nil {
		t.Fatalf("lp2 setLimit: %v", err)
	}

	// both claims are in flight -> the site discounts the combined draw
	if got, want := ledger.inflight(), 2*draw; got != want {
		t.Fatalf("combined inflight = %v, want %v", got, want)
	}
}
