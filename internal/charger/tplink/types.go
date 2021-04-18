package tplink

// TP-Link smart power plug/outlet responses
// https://www.softscheck.com/en/reverse-engineering-tp-link-hs110/#Portscan

// SystemResponse is the TP-Link plug/outlet api system response
type SystemResponse struct {
	System struct {
		SetRelayState struct {
			ErrCode int `json:"err_code,omitempty"`
		} `json:"set_relay_state"`
		GetSysinfo struct {
			ErrCode    int    `json:"err_code,omitempty"`
			SwVer      string `json:"sw_ver,omitempty"`
			Model      string `json:"model,omitempty"`
			Alias      string `json:"alias,omitempty"`
			DevName    string `json:"dev_name,omitempty"`
			RelayState int    `json:"relay_state,omitempty"`
			Feature    string `json:"feature,omitempty"`
		} `json:"get_sysinfo"`
	} `json:"system"`
}

// EmeterResponse is the TP-Link plug/outlet api emeter response
type EmeterResponse struct {
	Emeter struct {
		GetRealtime struct {
			Current float64 `json:"current,omitempty"`
			Voltage float64 `json:"voltage,omitempty"`
			Power   float64 `json:"power,omitempty"`
			Total   float64 `json:"total,omitempty"`
			ErrCode int     `json:"err_code,omitempty"`
		} `json:"get_realtime"`
	} `json:"emeter"`
}
