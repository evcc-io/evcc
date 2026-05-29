// Package ui persists UI-only state (color overrides, etc.).
package ui

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
)

// DeviceColor is the MQTT-safe publish format (map keys would leak into topic segments).
type DeviceColor struct {
	Title string `json:"title"`
	Color string `json:"color"`
}

var colorsMu sync.RWMutex

func GetDeviceColors() map[string]string {
	colorsMu.RLock()
	defer colorsMu.RUnlock()
	m := map[string]string{}
	_ = settings.Json(keys.DeviceColors, &m)
	return m
}

func SaveDeviceColors(m map[string]string) error {
	colorsMu.Lock()
	defer colorsMu.Unlock()
	clean := make(map[string]string, len(m))
	maps.Copy(clean, m)
	return settings.SetJson(keys.DeviceColors, clean)
}

func DeviceColorList() []DeviceColor {
	m := GetDeviceColors()
	list := make([]DeviceColor, 0, len(m))
	for title, color := range m {
		list = append(list, DeviceColor{Title: title, Color: color})
	}
	slices.SortFunc(list, func(a, b DeviceColor) int { return strings.Compare(a.Title, b.Title) })
	return list
}
