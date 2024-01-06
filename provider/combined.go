package provider

type combinedProvider struct {
	status func() (string, error)
}

func init() {
	registry.Add("combined", NewCombinedFromConfig)
	registry.Add("openwb", NewCombinedFromConfig)
}

// NewCombinedFromConfig creates combined provider
func NewCombinedFromConfig(other map[string]interface{}) (Provider, error) {
	status, err := NewOpenWBStatusProviderFromConfig(other)
	if err != nil {
		return nil, err
	}

	o := &combinedProvider{status: status}
	return o, nil
}

var _ StringProvider = (*combinedProvider)(nil)

func (o *combinedProvider) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		return o.status()
	}, nil
}
