package plugchoice

// StatusResponse is the connector status response
type StatusResponse struct {
	Data struct {
		UUID            string `json:"uuid"`
		ID              int    `json:"id"`
		Identity        string `json:"identity"`
		Reference       string `json:"reference"`
		ConnectionStatus string `json:"connection_status"`
		Status          string `json:"status"`
		Error           string `json:"error"`
		ErrorInfo       any    `json:"error_info"`
		CreatedAt       string `json:"created_at"`
		UpdatedAt       string `json:"updated_at"`
		Model           struct {
			Vendor string `json:"vendor"`
			Name   string `json:"name"`
		} `json:"model"`
		Connectors []Connector `json:"connectors"`
	} `json:"data"`
}

// Connector represents a charging connector
type Connector struct {
	ID          int    `json:"id"`
	ChargerID   int    `json:"charger_id"`
	ConnectorID int    `json:"connector_id"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	ErrorInfo   any    `json:"error_info"`
	MaxAmperage int    `json:"max_amperage"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// PowerResponse is the power usage response
type PowerResponse struct {
	Timestamp string `json:"timestamp"`
	KW        string `json:"kW"`
	L1        string `json:"L1"`
	L2        string `json:"L2"`
	L3        string `json:"L3"`
}