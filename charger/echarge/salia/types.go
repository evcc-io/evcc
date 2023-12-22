package salia

const (
	HeartBeat        = "salia/heartbeat"
	ChargeMode       = "salia/chargemode"
	PauseCharging    = "salia/pausecharging"
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
		ChargeMode    string
		PauseCharging int `json:"pausecharging,string"`
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
}
