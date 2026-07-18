package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestGetPlanUsesSharedPlan(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.planner = planner.New(util.NewLogger("foo"), nil)

	now := time.Now()
	shared := api.Rates{{Start: now, End: now.Add(time.Hour)}}
	lp.setSharedPlan(shared)

	// GetPlan returns the site-assigned shared plan verbatim
	assert.Equal(t, shared, lp.GetPlan(now.Add(2*time.Hour), time.Hour, 0, false))

	// cleared -> falls back to the independent path (simple plan, not the shared one)
	lp.setSharedPlan(nil)
	assert.NotEqual(t, shared, lp.GetPlan(now.Add(2*time.Hour), time.Hour, 0, false))
}

func TestSharedPlanRequestNoGoal(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	_, ok := lp.sharedPlanRequest()
	assert.False(t, ok, "no plan time -> no shared request")
}

func TestComputeSharedPlansWithoutCircuit(t *testing.T) {
	lp1 := NewLoadpoint(util.NewLogger("foo"), nil)
	lp2 := NewLoadpoint(util.NewLogger("foo"), nil)
	site := &Site{loadpoints: []*Loadpoint{lp1, lp2}}

	now := time.Now()
	site.computeSharedPlans(api.Rates{{Start: now, End: now.Add(time.Hour)}})

	// no circuit -> no contention -> independent planning (no shared plan set)
	assert.Nil(t, lp1.sharedPlan)
	assert.Nil(t, lp2.sharedPlan)
}
