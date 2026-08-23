package tariff

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// schedule is squashed into a tariff's config struct so every tariff supports
// the same interval option: a duration (e.g. "15m") or a cron expression
// (e.g. "15 0 * * *").
type schedule struct {
	Interval string
}

// refreshTimer drives a tariff's refresh loop. A nil sched means interval-based.
type refreshTimer struct {
	interval time.Duration
	sched    cron.Schedule
	stop     chan struct{}
}

func (s schedule) timer(fallback time.Duration) (*refreshTimer, error) {
	t := &refreshTimer{interval: fallback}
	if s.Interval == "" {
		return t, nil
	}

	if d, err := time.ParseDuration(s.Interval); err == nil {
		if d <= 0 {
			return nil, fmt.Errorf("interval: %q must be positive", s.Interval)
		}
		t.interval = d
		return t, nil
	}

	// local time, optionally overridden per-expression with a CRON_TZ= prefix
	sched, err := cron.ParseStandard(s.Interval)
	if err != nil {
		return nil, fmt.Errorf("interval: %q is neither a duration nor a cron expression: %w", s.Interval, err)
	}
	// a syntactically valid but impossible schedule (e.g. "0 0 30 2 *") yields
	// a zero next-fire; reject it so C() cannot busy-loop
	if sched.Next(time.Now()).IsZero() {
		return nil, fmt.Errorf("interval: cron %q never matches", s.Interval)
	}
	t.sched = sched
	return t, nil
}

// C returns a ticker-like channel so run loops keep the existing
// `for tick := …; ; <-tick` shape as a drop-in replacement for time.Tick.
// Call Stop when the loop exits to release the driving goroutine and timer.
func (t *refreshTimer) C() <-chan time.Time {
	ch := make(chan time.Time, 1)
	t.stop = make(chan struct{})
	stop := t.stop

	if t.sched == nil {
		go func() {
			ticker := time.NewTicker(t.interval)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case tt := <-ticker.C:
					select {
					case ch <- tt:
					case <-stop:
						return
					}
				}
			}
		}()
		return ch
	}

	go func() {
		for {
			next := t.sched.Next(time.Now())
			if next.IsZero() {
				return
			}
			timer := time.NewTimer(time.Until(next))
			select {
			case <-stop:
				timer.Stop()
				return
			case tt := <-timer.C:
				select {
				case ch <- tt:
				case <-stop:
					return
				}
			}
		}
	}()
	return ch
}

// Stop releases the goroutine and timer started by C. Safe to call once per C.
func (t *refreshTimer) Stop() {
	if t.stop != nil {
		close(t.stop)
	}
}

// window returns a staleness window for sizing util.Monitor, mirroring the
// former 2*interval behaviour. For cron it spans two scheduled intervals,
// measured between consecutive fires so it doesn't depend on when evcc starts.
func (t *refreshTimer) window() time.Duration {
	if t.sched == nil {
		return 2 * t.interval
	}
	next := t.sched.Next(time.Now())
	return 2 * t.sched.Next(next).Sub(next)
}

func (t *refreshTimer) stale(updated time.Time) bool {
	if t.sched == nil {
		return time.Since(updated) > t.interval
	}
	return time.Now().After(t.sched.Next(updated))
}
