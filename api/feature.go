package api

type Feature int

func (f *Feature) UnmarshalText(text []byte) error {
	feat, err := FeatureString(string(text))
	if err == nil {
		*f = feat
	}
	return err
}

//go:generate go run github.com/dmarkham/enumer@v1.5.8 -type Feature
const (
	_ Feature = iota
	Offline
	CoarseCurrent
	IntegratedDevice
	Heating
)
