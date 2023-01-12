package tasmota

// StatusResponse is the Status part of the Tasmota Status 0 command response
// https://tasmota.github.io/docs/JSON-Status-Responses/
type StatusResponse struct {
	Status struct {
		Module       int
		DeviceName   string
		FriendlyName []string
		Topic        string
		ButtonTopic  string
		Power        int
		PowerOnState int
		LedState     int
		LedMask      string
		SaveData     int
		SaveState    int
		SwitchTopic  string
		SwitchMode   []int
		ButtonRetain int
		SwitchRetain int
		SensorRetain int
		PowerRetain  int
		InfoRetain   int
		StateRetain  int
	}
}

// StatusSTSResponse is the StatusSTS part of the Tasmota Status 0 command response
// https://tasmota.github.io/docs/JSON-Status-Responses/
type StatusSTSResponse struct {
	StatusSTS struct {
		Power  string // ON, OFF, Error
		Power1 string // ON, OFF, Error
		Power2 string // ON, OFF, Error
		Power3 string // ON, OFF, Error
		Power4 string // ON, OFF, Error
		Power5 string // ON, OFF, Error
		Power6 string // ON, OFF, Error
		Power7 string // ON, OFF, Error
		Power8 string // ON, OFF, Error
	}
}

// PowerResponse is the Tasmota Power command Status response
// https://tasmota.github.io/docs/Commands/#with-web-requests
type PowerResponse struct {
	Power  string // ON, OFF, Error
	Power1 string // ON, OFF, Error
	Power2 string // ON, OFF, Error
	Power3 string // ON, OFF, Error
	Power4 string // ON, OFF, Error
	Power5 string // ON, OFF, Error
	Power6 string // ON, OFF, Error
	Power7 string // ON, OFF, Error
	Power8 string // ON, OFF, Error
}

// StatusSNSResponse is the Tasmota Status 8 command Status response
// https://tasmota.github.io/docs/JSON-Status-Responses/
type StatusSNSResponse struct {
	StatusSNS struct {
		Time string

		// Energy readings
		Energy struct {
			TotalStartTime string
			Total          float64
			Yesterday      float64
			Today          float64
			Power          interface{}
			ApparentPower  interface{}
			ReactivePower  interface{}
			Factor         interface{}
			Frequency      int
			Voltage        int
			Current        interface{}
		}

		// SML sensor readings
		SML struct {
			TotalIn   float64 `json:"total_in"`
			TotalOut  float64 `json:"total_out"`
			PowerCurr int     `json:"power_curr"`
		}
	}
}
