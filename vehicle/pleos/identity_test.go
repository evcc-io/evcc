package pleos

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestIdentityShared(t *testing.T) {
	log := util.NewLogger("foo")

	ts := NewIdentity(log, "kia", "id", "secret")
	assert.Same(t, ts, NewIdentity(log, "kia", "id", "secret"), "same api key")
	assert.NotSame(t, ts, NewIdentity(log, "kia", "other", "secret"), "other api key")
	assert.NotSame(t, ts, NewIdentity(log, "hyundai", "id", "secret"), "other brand")
}
