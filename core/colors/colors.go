// Package colors persists user-assigned device color overrides keyed by title.
package colors

import (
	"maps"
	"sync"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
)

var mu sync.RWMutex

// Get returns the persisted title→hex map (never nil).
func Get() map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	var m map[string]string
	_ = settings.Json(keys.DeviceColors, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

// Save persists the title→hex map.
func Save(m map[string]string) error {
	mu.Lock()
	defer mu.Unlock()
	clean := make(map[string]string, len(m))
	maps.Copy(clean, m)
	return settings.SetJson(keys.DeviceColors, clean)
}
