package smaevcharger

// SMA EV Charger 22 - json Responses

//Constants
const (
	ConstNConNCarNChar = float32(200111) // No Car connectec and no charging
	ConstYConYCarNChar = float32(200112) // Car connected and no charging
	ConstYConYCarYChar = float32(200113) // Car connected and charging

	ConstFastCharge = "4718" // Schnellladen - 4718
	ConstOptiCharge = "4719" // Optimiertes Laden - 4719
	ConstPlanCharge = "4720" // Laden mit Vorgabe - 4720
	ConstStopCharge = "4721" // Ladestopp - 4721

	ConstSwitchOeko = float32(4950) // Switch in PV Loading (Can be Optimized or Planned PV loading)
	ConstSwitchFast = float32(4718) // Switch in Fast Charge Mode

	ConstSendParameterFormat = "2006-01-02T15:04:05.000Z"
)

// Measurements Data json Response structure
type Measurements struct {
	ChannelId   string `json:"channelId"`
	ComponentId string `json:"componentId"`
	Values      []struct {
		Time  string  `json:"time"`
		Value float32 `json:"value"`
	} `json:"values"`
}

// Parameter Data json Response structure
type Parameters struct {
	ComponentId string `json:"componentId"`
	Values      []struct {
		ChannelId      string   `json:"channelId"`
		Editable       bool     `json:"editable"`
		PossibleValues []string `json:"possibleValues,omitempty"`
		State          string   `json:"state"`
		Timestamp      string   `json:"timestamp"`
		Value          string   `json:"value"`
	} `json:"values"`
}

// Parameter Data json Send structure
type SendParameter struct {
	Values []SendData `json:"values"`
}

// part of Paramter Send structure
type SendData struct {
	Timestamp string `json:"timestamp"`
	ChannelId string `json:"channelId"`
	Value     string `json:"value"`
}
