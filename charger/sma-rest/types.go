package smarest

// SMA EV Charger 22 - json Responses

const (
	MinAcceptedVersion = "1.2.23"
	TimestampFormat    = "2006-01-02T15:04:05.000Z"
)

type ErrUnknownMeasurement struct {
	Measurement string
}

func (e *ErrUnknownMeasurement) Error() string {
	return "unknown measurement: " + e.Measurement
}

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
	ComponentId string      `json:"componentId"`
	Values      []Parameter `json:"values"`
}

// Parameter is a single parameter including its metadata
type Parameter struct {
	ChannelId      string   `json:"channelId"`
	Editable       bool     `json:"editable"`
	PossibleValues []string `json:"possibleValues,omitempty"`
	Min            *float64 `json:"min,omitempty"`
	Max            *float64 `json:"max,omitempty"`
	State          string   `json:"state"`
	Timestamp      string   `json:"timestamp"`
	Value          string   `json:"value"`
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
