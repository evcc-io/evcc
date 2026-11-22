package fnn

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/core/site"
)

func decodeSignals(raw string) (map[string]bool, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, fmt.Errorf("signals: %w", err)
	}
	values := make(map[string]bool, len(data))
	for key, value := range data {
		switch key {
		case "w3", "w4", "s1", "s2":
		default:
			return nil, fmt.Errorf("signals: unknown contact %s", key)
		}
		value = bytes.TrimSpace(value)
		if !bytes.Equal(value, []byte("true")) && !bytes.Equal(value, []byte("false")) {
			return nil, fmt.Errorf("signals: %s must be boolean", key)
		}
		values[key] = bytes.Equal(value, []byte("true"))
	}
	if _, w3 := values["w3"]; !w3 {
		if _, s1 := values["s1"]; s1 {
			return nil, errors.New("signals: S1 requires W3")
		}
		if _, s2 := values["s2"]; s2 {
			return nil, errors.New("signals: S2 requires W3")
		}
		if _, w4 := values["w4"]; !w4 {
			return nil, errors.New("signals: must contain W3 or W4")
		}
	}
	return values, nil
}

func newSnapshotFnn(site site.API, dimPower, productionPower float64, getter func() (string, error), interval time.Duration) (*Fnn, error) {
	raw, err := getter()
	if err != nil {
		return nil, err
	}
	values, err := decodeSignals(raw)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]bool, len(values))
	for key := range values {
		keys[key] = true
	}
	contact := func(key string) func() (bool, error) {
		if !keys[key] {
			return nil
		}
		return func() (bool, error) { return values[key], nil }
	}
	c, err := NewFnn(site, dimPower, productionPower, contact("w3"), contact("s1"), contact("s2"), contact("w4"), interval)
	if err != nil {
		return nil, err
	}
	c.readSignals = func() error {
		raw, err := getter()
		if err != nil {
			return err
		}
		next, err := decodeSignals(raw)
		if err != nil {
			return err
		}
		if len(next) != len(keys) {
			return errors.New("signals: configured contacts changed")
		}
		for key := range keys {
			if _, ok := next[key]; !ok {
				return errors.New("signals: configured contacts changed")
			}
		}
		// Only the control loop accesses the snapshot; exported status reads limits.
		values = next
		return nil
	}
	return c, nil
}
