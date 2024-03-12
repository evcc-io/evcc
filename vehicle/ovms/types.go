package ovms

type StatusResponse struct {
	Unit     string  `json:"unit"`
	Odometer float64 `json:"odometer,string"`
}

type ChargeResponse struct {
	ChargeEtrFull    int64   `json:"charge_etr_full,string"`
	ChargeState      string  `json:"chargestate"`
	ChargePortOpen   int     `json:"cp_dooropen"`
	EstimatedRange   int64   `json:"estimatedrange,string"`
	MessageAgeServer int     `json:"m_msgage_s"`
	Soc              float64 `json:"soc,string"`
}

type LocationResponse struct {
	Latitude  float64 `json:"latitude,string"`
	Longitude float64 `json:"longitude,string"`
}

type ConnectResponse struct {
	NetConnected int `json:"v_net_connected"`
}
