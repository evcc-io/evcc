package tariff

import "github.com/evcc-io/evcc/api"

type proxyError struct {
	error
}

var _ api.Tariff = (*proxyError)(nil)

func (t *proxyError) Rates() (api.Rates, error) {
	return api.Rates{}, t.error
}

func (t *proxyError) Type() api.TariffType {
	return 0 // unknown
}
