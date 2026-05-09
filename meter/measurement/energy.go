package measurement

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Energy struct {
	Power  plugin.Config
	Import *plugin.Config // optional
	Export *plugin.Config // optional
	// Energy is a legacy alias used when neither Import nor Export is set.
	// For meters, the same value is wired to both interfaces and the site role
	// (pv/battery → Export; grid/aux/ext → Import) decides which one is read.
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

	// legacy: a single energy field is exposed as both import and export;
	// the site role decides which interface is used.
	importCfg := cc.Import
	exportCfg := cc.Export
	if cc.Energy != nil {
		if importCfg == nil {
			importCfg = cc.Energy
		}
		if exportCfg == nil {
			exportCfg = cc.Energy
		}
	}

	importG, err := importCfg.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("import: %w", err)
	}

	exportG, err := exportCfg.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("export: %w", err)
	}

	return powerG, importG, exportG, nil
}
