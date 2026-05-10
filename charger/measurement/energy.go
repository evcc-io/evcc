package measurement

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/plugin"
)

type Energy struct {
	Power  *plugin.Config
	Import *plugin.Config // optional
	Export *plugin.Config // optional
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

	importG, err := cc.Import.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("import: %w", err)
	}

	exportG, err := cc.Export.FloatGetter(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("export: %w", err)
	}

	return powerG, importG, exportG, nil
}

// AliasImport assigns the legacy energy field onto Import
func (cc *Energy) AliasImport(energy *plugin.Config) error {
	if energy == nil {
		return nil
	}
	if cc.Import != nil {
		return errors.New("energy and import/export are mutually exclusive")
	}
	cc.Import = energy
	return nil
}
