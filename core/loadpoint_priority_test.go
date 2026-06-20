package core

import (
	"errors"
	"time"

	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// memSettings is a minimal in-memory settings.Settings implementation for tests.
type memSettings struct {
	data map[string]any
}

func newMemSettings() *memSettings {
	return &memSettings{data: make(map[string]any)}
}

func (s *memSettings) SetString(key string, val string)     { s.data[key] = val }
func (s *memSettings) SetInt(key string, val int64)         { s.data[key] = val }
func (s *memSettings) SetFloat(key string, val float64)     { s.data[key] = val }
func (s *memSettings) SetFloatPtr(key string, val *float64) { s.data[key] = val }
func (s *memSettings) SetTime(key string, val time.Time)    { s.data[key] = val }
func (s *memSettings) SetBool(key string, val bool)         { s.data[key] = val }
func (s *memSettings) SetJson(key string, val any) error    { s.data[key] = val; return nil }

func (s *memSettings) String(key string) (string, error) {
	if v, ok := s.data[key].(string); ok {
		return v, nil
	}
	return "", errors.New("not found")
}

func (s *memSettings) Int(key string) (int64, error) {
	if v, ok := s.data[key].(int64); ok {
		return v, nil
	}
	return 0, errors.New("not found")
}

func (s *memSettings) Float(key string) (float64, error) {
	if v, ok := s.data[key].(float64); ok {
		return v, nil
	}
	return 0, errors.New("not found")
}

func (s *memSettings) Time(key string) (time.Time, error) {
	if v, ok := s.data[key].(time.Time); ok {
		return v, nil
	}
	return time.Time{}, errors.New("not found")
}

func (s *memSettings) Bool(key string) (bool, error) {
	if v, ok := s.data[key].(bool); ok {
		return v, nil
	}
	return false, errors.New("not found")
}

func (s *memSettings) Json(key string, res any) error { return errors.New("not implemented") }

func newPriorityTestLoadpoint() (*Loadpoint, *memSettings) {
	s := newMemSettings()
	lp := NewLoadpoint(util.NewLogger("foo"), s)
	return lp, s
}

func TestSetPriorityStrategy(t *testing.T) {
	lp, s := newPriorityTestLoadpoint()

	// valid: soc
	lp.SetPriorityStrategy(api.PrioritySoc)
	assert.Equal(t, api.PrioritySoc, lp.GetPriorityStrategy())
	assert.Equal(t, string(api.PrioritySoc), s.data[keys.PriorityStrategy], "soc must be persisted")

	// valid: deficit
	lp.SetPriorityStrategy(api.PriorityDeficit)
	assert.Equal(t, api.PriorityDeficit, lp.GetPriorityStrategy())
	assert.Equal(t, string(api.PriorityDeficit), s.data[keys.PriorityStrategy])

	// "static" normalizes to the empty PriorityStatic value
	lp.SetPriorityStrategy(api.PriorityStrategy("static"))
	assert.Equal(t, api.PriorityStatic, lp.GetPriorityStrategy())
	assert.Equal(t, string(api.PriorityStatic), s.data[keys.PriorityStrategy])

	// invalid: rejected, state unchanged
	lp.SetPriorityStrategy(api.PrioritySoc)
	delete(s.data, keys.PriorityStrategy)
	lp.SetPriorityStrategy(api.PriorityStrategy("bogus"))
	assert.Equal(t, api.PrioritySoc, lp.GetPriorityStrategy(), "invalid strategy must not change state")
	_, persisted := s.data[keys.PriorityStrategy]
	assert.False(t, persisted, "invalid strategy must not be persisted")
}

func TestSetPriorityHysteresis(t *testing.T) {
	lp, s := newPriorityTestLoadpoint()

	// valid
	lp.SetPriorityHysteresis(5)
	assert.Equal(t, 5, lp.GetPriorityHysteresis())
	assert.Equal(t, int64(5), s.data[keys.PriorityHysteresis], "valid hysteresis must be persisted")

	// boundary: 99 ok
	lp.SetPriorityHysteresis(99)
	assert.Equal(t, 99, lp.GetPriorityHysteresis())
	assert.Equal(t, int64(99), s.data[keys.PriorityHysteresis])

	// boundary: 0 ok (off)
	lp.SetPriorityHysteresis(0)
	assert.Equal(t, 0, lp.GetPriorityHysteresis())
	assert.Equal(t, int64(0), s.data[keys.PriorityHysteresis])

	// invalid: > 99 rejected, state unchanged
	lp.SetPriorityHysteresis(7)
	delete(s.data, keys.PriorityHysteresis)
	lp.SetPriorityHysteresis(100)
	assert.Equal(t, 7, lp.GetPriorityHysteresis(), "out-of-range hysteresis must not change state")
	_, persisted := s.data[keys.PriorityHysteresis]
	assert.False(t, persisted, "out-of-range hysteresis must not be persisted")

	// invalid: negative rejected
	lp.SetPriorityHysteresis(-1)
	assert.Equal(t, 7, lp.GetPriorityHysteresis(), "negative hysteresis must not change state")
}
