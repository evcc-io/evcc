package iotawatt

// Series describes an available IoTaWatt input or output series.
type Series struct {
	Name string `json:"name"`
	Unit string `json:"unit"`
}

// ShowSeriesResponse is the response from the query?show=series endpoint.
type ShowSeriesResponse struct {
	Series []Series `json:"series"`
}

// DeviceConfig is the IoTaWatt device configuration from /config.txt.
type DeviceConfig struct {
	Inputs    []*InputConfig `json:"inputs"`
	Outputs   []*OutputConfig `json:"outputs"`
	Derive3Ph bool           `json:"derive3ph"`
}

// InputConfig describes an input channel in the device config.
type InputConfig struct {
	Channel int     `json:"channel"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`    // "VT" or "CT"
	VPhase  float64 `json:"vphase"`  // 0=L1, 120=L2, 240=L3
}

// OutputConfig describes an output in the device config.
type OutputConfig struct {
	Name  string `json:"name"`
	Units string `json:"units"`
}

// Phase returns the phase number (1, 2, or 3) based on VPhase.
func (c *InputConfig) Phase() int {
	switch {
	case c.VPhase >= 100 && c.VPhase < 140:
		return 2
	case c.VPhase >= 220:
		return 3
	default:
		return 1
	}
}
