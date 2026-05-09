package measurement

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Energy struct {
	Power  *plugin.Config // optional
	Import *plugin.Config // optional
	Export *plugin.Config // optional
	// Energy is a legacy alias for Import. Chargers always import energy, so
	// the alias only feeds the Import getter (unlike meters, which alias to both).
	Energy *plugin.Config
}

func (cc *Energy) Configure(ctx context.Context) (
	func() (float64, error),
	func() (float64, error),
	func() (float64, error),
	error,
) {
	powerG, err := cc.Power.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("power: %w", err)
	}

	// legacy: chargers always import, so 'energy' aliases to import only.
	importCfg := cc.Import
	if cc.Energy != nil && importCfg == nil {
		importCfg = cc.Energy
	}

	importG, err := importCfg.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("import: %w", err)
	}

	exportG, err := cc.Export.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("export: %w", err)
	}

	return powerG, importG, exportG, nil
}
