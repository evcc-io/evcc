package salia

type Api struct {
	Device struct{}
	Secc   Secc
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
		ChargeMode string
	}
	Metering struct {
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
}
