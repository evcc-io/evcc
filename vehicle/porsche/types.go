package porsche

type Vehicle struct {
	VIN              string
	ModelDescription string
	Pictures         []struct {
		URL         string
		View        string
		Size        int
		Width       int
		Height      int
		Transparent bool
	}
}

type VehiclePairingResponse struct {
	VIN                string
	PairingCode        string
	Status             string
	CanSendPairingCode bool
}

type StatusResponse struct {
	BatteryLevel struct {
		Unit  string
		Value float64
	}
	Mileage struct {
		Unit  string
		Value float64
	}
	RemainingRanges struct {
		ElectricalRange struct {
			Distance struct {
				Unit  string
				Value float64
			}
		}
	}
}

type CapabilitiesResponse struct {
	DisplayParkingBrake      bool
	NeedsSPIN                bool
	HasRDK                   bool
	EngineType               string
	CarModel                 string
	OnlineRemoteUpdateStatus struct {
		EditableByUser bool
		Active         bool
	}
	HeatingCapabilities struct {
		FrontSeatHeatingAvailable bool
		RearSeatHeatingAvailable  bool
	}
	SteeringWheelPosition string
	HasHonkAndFlash       bool
}

type EmobilityResponse struct {
	BatteryChargeStatus *struct {
		ChargeRate struct {
			Unit             string
			Value            float64
			ValueInKmPerHour int64
		}
		ChargingInDCMode                            bool
		ChargingMode                                string
		ChargingPower                               float64
		ChargingReason                              string
		ChargingState                               string
		ChargingTargetDateTime                      string
		ExternalPowerSupplyState                    string
		PlugState                                   string
		RemainingChargeTimeUntil100PercentInMinutes int64
		StateOfChargeInPercentage                   int64
		RemainingERange                             struct {
			OriginalUnit      string
			OriginalValue     int64
			Unit              string
			Value             int64
			ValueInKilometers int64
		}
	}
	ChargingStatus string
	DirectCharge   struct {
		Disabled bool
		IsActive bool
	}
	DirectClimatisation struct {
		ClimatisationState         string
		RemainingClimatisationTime int64
	}
	PcckErrorMessage string
}
