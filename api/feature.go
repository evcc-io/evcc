package api

type Feature int

func (f *Feature) UnmarshalText(text []byte) error {
	feat, err := FeatureString(string(text))
	if err == nil {
		*f = feat
	}
	return err
}

//go:generate enumer -type Feature
const (
	_ Feature = iota

	Hidden           // vehicle: hidden from api
	Offline          // vehicle: no capacity/soc
	CoarseCurrent    // vehicle: 1A resolution
	IntegratedDevice // charger: no separate vehicle
	Heating          // charger: heating device
)
