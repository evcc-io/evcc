package provider

import "context"

type combinedProvider struct {
	status func() (string, error)
}

func init() {
	registry.AddCtx("combined", NewCombinedFromConfig)
	registry.AddCtx("openwb", NewCombinedFromConfig)
}

// NewCombinedFromConfig creates combined provider
func NewCombinedFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	status, err := NewOpenWBStatusProviderFromConfig(ctx, other)
	if err != nil {
		return nil, err
	}

	o := &combinedProvider{status: status}
	return o, nil
}

var _ StringProvider = (*combinedProvider)(nil)

func (o *combinedProvider) StringGetter() (func() (string, error), error) {
	return o.status, nil
}
