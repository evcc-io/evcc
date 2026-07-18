package aa55

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPacerGatePerInverter(t *testing.T) {
	p := &pacer{gates: make(map[string]*inverterGate)}

	g := p.gate("1.2.3.4:8899")
	assert.Same(t, g, p.gate("1.2.3.4:8899"), "same inverter shares one gate")
	assert.NotSame(t, g, p.gate("5.6.7.8:8899"), "different inverters get separate gates")
}

func TestGateSpacing(t *testing.T) {
	g := &inverterGate{}
	const delay = 30 * time.Millisecond

	g.wait(delay) // first send: no prior, returns immediately
	start := time.Now()
	g.wait(delay) // second send: must wait ~delay since the first

	assert.GreaterOrEqual(t, time.Since(start), delay/2, "second send is spaced from the first")
}
