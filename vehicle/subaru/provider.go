package subaru

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// retryTimeout bounds how long incomplete status payloads are retried before giving up
const retryTimeout = 2 * time.Minute

var errIncompleteStatus = errors.New("incomplete status payload")

type Provider struct {
	status       func() (Status, error)
	incompleteAt time.Time
}

func NewProvider(a *API, vin string, cache time.Duration) *Provider {
	impl := &Provider{}
	impl.status = util.Cached(func() (Status, error) {
		res, err := a.Status(vin)
		if err != nil {
			return res, err
		}
		return res, impl.validate(res)
	}, cache)
	return impl
}

// validate returns ErrMustRetry while the payload is incomplete and ErrTimeout once retryTimeout has elapsed
func (v *Provider) validate(res Status) error {
	if !incomplete(res) {
		v.incompleteAt = time.Time{}
		return nil
	}

	if v.incompleteAt.IsZero() {
		v.incompleteAt = time.Now()
	}

	if time.Since(v.incompleteAt) > retryTimeout {
		return fmt.Errorf("%w: %w", errIncompleteStatus, api.ErrTimeout)
	}

	return fmt.Errorf("%w: %w", errIncompleteStatus, api.ErrMustRetry)
}

func incomplete(res Status) bool {
	return res.Payload.EvRangeWithAc.Unit == "" ||
		(res.Payload.BatteryLevel == 0 && res.Payload.EvRangeWithAc.Value == 0)
}

func (v *Provider) Soc() (float64, error) {
	res, err := v.status()
	return float64(res.Payload.BatteryLevel), err
}

func (v *Provider) Range() (int64, error) {
	res, err := v.status()
	if err != nil {
		return 0, err
	}
	return res.Payload.EvRangeWithAc.ValueInKilometers()
}
