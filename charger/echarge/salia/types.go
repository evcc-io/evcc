package salia

import (
	"encoding/json"
	"strconv"
)

const (
	HeartBeat        = "salia/heartbeat"
	ChargeMode       = "salia/chargemode"
	PauseCharging    = "salia/pausecharging"
	SetPhase         = "salia/phase_switching/setphase"
	GridCurrentLimit = "grid_current_limit"
)

type Api struct {
	Device struct {
		ModelName       string
		SoftwareVersion string `json:"software_version"`
	}
	Secc Secc
}

type Secc struct {
	Port0 Port
}

type Port struct {
	Ci struct {
		Evse struct {
			Basic struct {
				OfferedCurrentLimit float64 `json:"offered_current_limit,string"`
			}
		}
		Charge struct {
			Cp struct {
				Status string
			}
		}
	}
	Salia struct {
		ChargeMode     string
		PauseCharging  int `json:"pausecharging,string"`
		PhaseSwitching struct {
			Actual   int `json:",string"`
			Status   string
			SetPhase string `json:"setphase,omitempty"`
		} `json:"phase_switching"`
	}
	Metering struct {
		Meter struct {
			Type      string
			Available int `json:",string"`
		}
		Power struct {
			ActiveTotal struct {
				Actual float64 `json:",string"`
			} `json:"active_total"`
			Active struct {
				AC struct {
					L1 struct {
						Actual float64 `json:",string"`
					}
					L2 struct {
						Actual float64 `json:",string"`
					}
					L3 struct {
						Actual float64 `json:",string"`
					}
				}
			}
		}
		Energy struct {
			ActiveImport struct {
				Actual float64 `json:",string"`
			} `json:"active_import"`
		}
		Current struct {
			AC struct {
				L1 struct {
					Actual float64 `json:",string"`
				}
				L2 struct {
					Actual float64 `json:",string"`
				}
				L3 struct {
					Actual float64 `json:",string"`
				}
			}
		}
	}
	EvPresent        int     `json:"ev_present,string"`
	Charging         int     `json:",string"`
	GridCurrentLimit float64 `json:"grid_current_limit,string"`
	Session          struct {
		AuthorizationStatus string `json:"authorization_status"`
		AuthorizationMethod string `json:"authorization_method"`
	} `json:"session"`
	RFID struct {
		Type                 string               `json:"type"`
		Available            string               `json:"available"`
		Authorizereq         string               `json:"authorizereq"`
		AuthorizationRequest AuthorizationRequest `json:"authorization_request"`
	} `json:"rfid"`
}

type AuthorizationRequest struct {
	Protocol string
	Key      string
}

func (a *AuthorizationRequest) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return nil
	}

	var arr []string
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return nil
	}

	if len(arr) == 2 {
		*a = AuthorizationRequest{
			Protocol: arr[0],
			Key:      arr[1],
		}
	}

	return nil
}
