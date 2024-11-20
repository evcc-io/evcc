package pipe

import (
	"runtime"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

func TestDeduplicator(t *testing.T) {
	l := NewDeduplicator(time.Hour, "filtered").(*Deduplicator)
	clck := clock.NewMock()
	l.clock = clck

	in := make(chan util.Param)
	out := l.Pipe(in)

	p := util.Param{Key: "k", Val: 1}
	in <- p

	if o := <-out; o.Key != p.Key || o.Val != p.Val {
		t.Errorf("unexpected param %v", o)
	}

	p = util.Param{Key: "k", Val: 2}
	in <- p

	if o := <-out; o.Key != p.Key || o.Val != p.Val {
		t.Errorf("unexpected param %v", o)
	}

	// allow nils
	p = util.Param{Key: "k", Val: nil}
	in <- p

	if o := <-out; o.Key != p.Key || o.Val != p.Val {
		t.Errorf("unexpected param %v", o)
	}

	p = util.Param{Key: "filtered", Val: 3}
	in <- p

	if o := <-out; o.Key != p.Key || o.Val != p.Val {
		t.Errorf("unexpected param %v", o)
	}

	p = util.Param{Key: "filtered", Val: 4}
	in <- p

	if o := <-out; o.Key != p.Key || o.Val != p.Val {
		t.Errorf("unexpected param %v", o)
	}

	// resend
	in <- p

	runtime.Gosched()
	select {
	case o := <-out:
		t.Errorf("unexpected param %v", o)
	case <-time.After(time.Millisecond):
	}

	// resend later
	clck.Add(2 * l.interval)
	in <- p

	if o := <-out; o.Key != p.Key || o.Val != p.Val {
		t.Errorf("unexpected param %v", o)
	}

	// allow nils
	p = util.Param{Key: "filtered", Val: nil}
	in <- p

	if o := <-out; o.Key != p.Key || o.Val != p.Val {
		t.Errorf("unexpected param %v", o)
	}
}
