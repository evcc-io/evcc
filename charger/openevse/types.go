package openevse

// ClientID is evcc's registered EVSE client id: vendor 0x0004 (evcc), client 0x0001.
// See EvseClient_evcc in the firmware's src/evse_man.h.
const ClientID = 0x00040001

const (
	Enabled  = "active"
	Disabled = "disabled"
)

// Status keeps the interesting properties of the /status document.
// The /ws endpoint sends the full document on connect and afterwards only
// the changed keys, so frames are merged into one Status by json.Unmarshal.
type Status struct {
	Amp           float64 `json:"amp"`            // charge current in mA
	Voltage       float64 `json:"voltage"`        // V
	Power         float64 `json:"power"`          // W, honours the firmware's is_threephase setting
	Pilot         float64 `json:"pilot"`          // pilot current in A
	Elapsed       float64 `json:"elapsed"`        // session duration in seconds
	SessionEnergy float64 `json:"session_energy"` // Wh
	TotalEnergy   float64 `json:"total_energy"`   // kWh
	State         int     `json:"state"`          // 1=A 2=B 3=C 4=D 5-11=F 254=sleeping 255=disabled
	Status        string  `json:"status"`         // active, disabled, none, unknown
	Vehicle       int     `json:"vehicle"`        // 0=not connected, 1=connected
	// ManualOverride is 1 when the firmware's manual override claim (priority 1000,
	// set by the charger's own UI/button, or a leftover from the previous evcc
	// driver) is active. It outranks evcc's claim (priority 500), so evcc's writes
	// have no effect until the override is cleared.
	ManualOverride int `json:"manual_override"`
	// RfidAuth is the RFID tag uid that authorised the current session, present
	// only when RFID is enabled on the charger. It is empty when no card has
	// authorised the session.
	RfidAuth string `json:"rfid_auth"`
}

// Claim is the body of POST /claims/{client}
type Claim struct {
	State         string `json:"state"`                    // active or disabled
	ChargeCurrent int    `json:"charge_current,omitempty"` // A, 0 = not set
}
