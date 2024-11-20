package smaevcharger

// SMA EV Charger 22 - json Responses

const (
	MinAcceptedVersion = "1.2.23"
	TimestampFormat    = "2006-01-02T15:04:05.000Z"

	StatusA       = float64(200111) // Not connected
	StatusB       = float64(200112) // Connected and not charging
	StatusC       = float64(200113) // Connected and charging
	ChargerLocked = float64(5169)   // Charger locked

	SwitchOeko = float64(4950) // Switch in PV Loading (Optimized or Planned PV loading)
	SwitchFast = float64(4718) // Switch in Fast Charge Mode

	FastCharge = "4718" // Schnellladen - 4718
	OptiCharge = "4719" // Optimiertes Laden - 4719
	PlanCharge = "4720" // Laden mit Vorgabe - 4720
	StopCharge = "4721" // Ladestopp - 4721

	ChargerAppLockEnabled  = "1129"
	ChargerAppLockDisabled = "1130"

	ChargerManualLockEnabled  = "5171"
	ChargerManualLockDisabled = "5172"
)

// Measurements Data json Response structure
type Measurements struct {
	ChannelId   string `json:"channelId"`
	ComponentId string `json:"componentId"`
	Values      []struct {
		Time  string  `json:"time"`
		Value float64 `json:"value"`
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
	Values []Value `json:"values"`
}

// part of Parameter Send structure
type Value struct {
	Timestamp string `json:"timestamp"`
	ChannelId string `json:"channelId"`
	Value     string `json:"value"`
}
