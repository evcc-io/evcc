package tasmota

import "encoding/json"

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
			Power          []float64
			ApparentPower  []float64
			ReactivePower  []float64
			Factor         []float64
			Frequency      int
			Voltage        int
			Current        []float64
		}

		// SML sensor readings
		SML struct {
			TotalIn   float64 `json:"total_in"`
			TotalOut  float64 `json:"total_out"`
			PowerCurr int     `json:"power_curr"`
		}
	}
}

func (v *StatusSNSResponse) UnmarshalJSON(data []byte) error {
	var err error
	// Single energy meter response
	var sr struct {
		StatusSNS struct {
			Time string

			// Energy readings
			Energy struct {
				TotalStartTime string
				Total          float64
				Yesterday      float64
				Today          float64
				Power          float64
				ApparentPower  float64
				ReactivePower  float64
				Factor         float64
				Frequency      int
				Voltage        int
				Current        float64
			}

			// SML sensor readings
			SML struct {
				TotalIn   float64 `json:"total_in"`
				TotalOut  float64 `json:"total_out"`
				PowerCurr int     `json:"power_curr"`
			}
		}
	}
	// Multi energy meter response
	var mr struct {
		StatusSNS struct {
			Time string

			// Energy readings
			Energy struct {
				TotalStartTime string
				Total          float64
				Yesterday      float64
				Today          float64
				Power          []float64
				ApparentPower  []float64
				ReactivePower  []float64
				Factor         []float64
				Frequency      int
				Voltage        int
				Current        []float64
			}

			// SML sensor readings
			SML struct {
				TotalIn   float64 `json:"total_in"`
				TotalOut  float64 `json:"total_out"`
				PowerCurr int     `json:"power_curr"`
			}
		}
	}

	//Try MULTI energy meter response
	if err = json.Unmarshal(data, &mr); err == nil {
		v.StatusSNS.Time = mr.StatusSNS.Time

		v.StatusSNS.Energy.TotalStartTime = mr.StatusSNS.Energy.TotalStartTime
		v.StatusSNS.Energy.Total = mr.StatusSNS.Energy.Total
		v.StatusSNS.Energy.Yesterday = mr.StatusSNS.Energy.Yesterday
		v.StatusSNS.Energy.Today = mr.StatusSNS.Energy.Today
		v.StatusSNS.Energy.Power = mr.StatusSNS.Energy.Power
		v.StatusSNS.Energy.ApparentPower = mr.StatusSNS.Energy.ApparentPower
		v.StatusSNS.Energy.ReactivePower = mr.StatusSNS.Energy.ReactivePower
		v.StatusSNS.Energy.Factor = mr.StatusSNS.Energy.Factor
		v.StatusSNS.Energy.Frequency = mr.StatusSNS.Energy.Frequency
		v.StatusSNS.Energy.Voltage = mr.StatusSNS.Energy.Voltage
		v.StatusSNS.Energy.Current = mr.StatusSNS.Energy.Current

		v.StatusSNS.SML.TotalIn = mr.StatusSNS.SML.TotalIn
		v.StatusSNS.SML.TotalOut = mr.StatusSNS.SML.TotalOut
		v.StatusSNS.SML.PowerCurr = mr.StatusSNS.SML.PowerCurr
	} else {
		//Try SINGLE energy meter response
		if err = json.Unmarshal(data, &sr); err == nil {
			v.StatusSNS.Time = sr.StatusSNS.Time

			v.StatusSNS.Energy.TotalStartTime = sr.StatusSNS.Energy.TotalStartTime
			v.StatusSNS.Energy.Total = sr.StatusSNS.Energy.Total
			v.StatusSNS.Energy.Yesterday = sr.StatusSNS.Energy.Yesterday
			v.StatusSNS.Energy.Today = sr.StatusSNS.Energy.Today
			v.StatusSNS.Energy.Power = append(v.StatusSNS.Energy.Power, sr.StatusSNS.Energy.Power)
			v.StatusSNS.Energy.ApparentPower = append(v.StatusSNS.Energy.ApparentPower, sr.StatusSNS.Energy.ApparentPower)
			v.StatusSNS.Energy.ReactivePower = append(v.StatusSNS.Energy.ReactivePower, sr.StatusSNS.Energy.ReactivePower)
			v.StatusSNS.Energy.Factor = append(v.StatusSNS.Energy.Factor, sr.StatusSNS.Energy.Factor)
			v.StatusSNS.Energy.Frequency = sr.StatusSNS.Energy.Frequency
			v.StatusSNS.Energy.Voltage = sr.StatusSNS.Energy.Voltage
			v.StatusSNS.Energy.Current = append(v.StatusSNS.Energy.Current, sr.StatusSNS.Energy.Current)

			v.StatusSNS.SML.TotalIn = sr.StatusSNS.SML.TotalIn
			v.StatusSNS.SML.TotalOut = sr.StatusSNS.SML.TotalOut
			v.StatusSNS.SML.PowerCurr = sr.StatusSNS.SML.PowerCurr
		}
	}

	return err
}
