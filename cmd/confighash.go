package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
)

// deviceKind returns the template name for template devices, otherwise the type
func deviceKind(typ string, other map[string]any) string {
	if strings.EqualFold(typ, "template") {
		for k, v := range other {
			if strings.EqualFold(k, "template") {
				if s, ok := v.(string); ok && s != "" {
					return s
				}
			}
		}
	}
	return typ
}

// dbKinds returns the device kinds configured in the database for a class
func dbKinds(class templates.Class) []string {
	if db.Instance == nil {
		return nil
	}
	configs, err := config.ConfigurationsByClass(class)
	if err != nil {
		return nil
	}
	kinds := make([]string, 0, len(configs))
	for _, c := range configs {
		n := c.Named()
		kinds = append(kinds, deviceKind(n.Type, n.Other))
	}
	return kinds
}

// classKinds collects the sorted device kinds for a class from both the static
// (yaml) config and the database
func classKinds(static []config.Named, class templates.Class) []string {
	kinds := make([]string, 0, len(static))
	for _, n := range static {
		kinds = append(kinds, deviceKind(n.Type, n.Other))
	}
	kinds = append(kinds, dbKinds(class)...)
	sort.Strings(kinds)
	return kinds
}

// configHash computes a deterministic fingerprint over the number and kind of
// configured meters, chargers, loadpoints and tariffs
func configHash(conf *globalconfig.All) string {
	tariffs := make([]string, 0)
	for _, t := range []config.Typed{conf.Tariffs.Grid, conf.Tariffs.FeedIn, conf.Tariffs.Co2, conf.Tariffs.Planner} {
		if t.Type != "" {
			tariffs = append(tariffs, deviceKind(t.Type, t.Other))
		}
	}
	for _, t := range conf.Tariffs.Solar {
		if t.Type != "" {
			tariffs = append(tariffs, deviceKind(t.Type, t.Other))
		}
	}
	tariffs = append(tariffs, dbKinds(templates.Tariff)...)
	sort.Strings(tariffs)

	data := map[string]any{
		"meters":     classKinds(conf.Meters, templates.Meter),
		"chargers":   classKinds(conf.Chargers, templates.Charger),
		"loadpoints": classKinds(conf.Loadpoints, templates.Loadpoint),
		"tariffs":    tariffs,
	}

	b, _ := json.Marshal(data)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
