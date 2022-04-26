package smaevcharger

// SMA EV Charger 22 - json Responses

// Auth Token Data json Response structure
type AuthToken struct {
	Access_token  string `json:"access_token"`
	Expires_in    int    `json:"expires_in"`
	Refresh_token string `json:"refresh_token"`
	Token_type    string `json:"token_type"`
	UiIdleTime    string `json:"uiIdleTime"`
}

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
