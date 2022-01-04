package api

type SimType int64

const (
	Sim_grid SimType = iota
	Sim_pv
	Sim_battery
	Sim_vehicle
	Sim_charger
	Sim_home
	Sim_switch1p3p
	Hil_switch1p3p // 1p3pswitch that supports the interfaces for hardware in the loop testing (connectable to the simulator)
)

type Sim interface {
	SimType() (SimType, error)
	SetName(name string) error
	Name() (string, error)
}

type SimPower interface {
	Meter
	SetCurrentPower(powerW float64) error // [W]
}

type SimBattery interface {
	Sim
	SimPower
	Battery
	SetSoC(soc float64) error                           // [%]
	UpdateSoC(availablePowerW float64) (float64, error) // [in] available power for charging/discharging, [out] power used for charging/discharging
	SetCapacity(capacitykWh int64) error                // [kWh]
	Capacity() int64                                    // [kWh]
	SetPowerLimit(powerLimitW float64) error            // [W]
	PowerLimit() (float64, error)                       // [W]
}

type SimMeter interface {
	Sim
	SimPower
}

type SimCharger interface {
	Sim
	Charger
	Meter
	Update() (float64, error)
	GetMaxCurrent() (int64, error) // [A]
	SetStatus(status ChargeStatus) error
	Connect(vehicle SimVehicle) error // connect a vehicle to the charger
	Disconnect() error                // disconnect a vehicle from the charger
	Switch1p3pName() (string, error)  // gives the name of the 1p3p switch, "" if no switch is configured
	SetSwitch1p3p(switch1p3p SimChargePhases) error
}

type SimChargePhases interface {
	Sim
	ChargePhases
	ChargeEnable
	GetPhases1p3p() (int, error)
}

type LockPhases1p3p interface {
	LockPhases1p3p(phases int) error
	UnlockPhases1p3p() error
}

type SimVehicle interface {
	SimBattery
	Battery
	SetTitle(title string) error
	SetIdentifiers(identifiers []string) error
	SetOnIdentified(action ActionConfig) error
	Bidirectional() (bool, error)
	SetBidirectional(isBidirectional bool) error
}
