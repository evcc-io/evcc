package aa55

import (
	"sync"
	"time"
)

// pace spaces sends per inverter. Each source dials its own UDP connection, so
// the gap is enforced per remote address at package level, not per connection.
var pace = &pacer{gates: make(map[string]*inverterGate)}

type pacer struct {
	mu    sync.Mutex
	gates map[string]*inverterGate
}

// inverterGate serializes exchanges to one inverter and tracks its last send.
type inverterGate struct {
	mu   sync.Mutex
	last time.Time
}

// gate returns the shared gate for the inverter at addr, creating it on first use.
func (p *pacer) gate(addr string) *inverterGate {
	p.mu.Lock()
	defer p.mu.Unlock()

	g := p.gates[addr]
	if g == nil {
		g = &inverterGate{}
		p.gates[addr] = g
	}
	return g
}

// wait blocks until delay has elapsed since the previous send to this inverter,
// then records the new send time. The caller must hold g.mu.
func (g *inverterGate) wait(delay time.Duration) {
	if d := time.Until(g.last.Add(delay)); d > 0 {
		time.Sleep(d)
	}
	g.last = time.Now()
}
