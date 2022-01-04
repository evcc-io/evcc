package simulation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type Actors struct {
	Meters       map[string]api.Meter
	Chargers     map[string]api.Charger
	Switches1p3p map[string]api.ChargePhases
	Vehicles     map[string]api.Vehicle
}

type SimulatorCfg struct {
	Active  bool         // true if the simulator shall be executed
	Program []SimCommand // the simulation program to execute
}

type SimCommand struct {
	Cmd       string // the command to execute
	Object    string // the object on which the command shall be executed
	Attribute string // the attribute of the object on which the command shall be executed
	Value     string // the value of the attribute / object that shall be set or tested
	Timeout   string // timeout for the command
}

type Simulator struct {
	log           *util.Logger
	cfg           *SimulatorCfg
	simsByType    map[api.SimType][]api.Sim
	simsByName    map[string]api.Sim
	progStep      int           // index of the next element to process in the Process array
	progStepStart time.Time     // start timestamp of the current process step
	lastProgCycle time.Time     // timestamp of the last update call
	progCycle     time.Duration // average time between two update calls
}

func init() {
	registry.Add("simulator", NewSimulatorFromConfig)
}

// NewProgSimFromConfig creates a programmable simulator from config
func NewSimulatorFromConfig(actors *Actors, other map[string]interface{}) (api.Updateable, error) {

	cfg := &SimulatorCfg{
		Active: false,
	}

	if err := util.DecodeOther(other, &cfg); err != nil {
		return nil, err
	}

	return NewSimulator(actors, cfg)
}

// NewSimulator creates a simulator with the given configuration
// if the simulator config says that it is active the simulation program is read
func NewSimulator(actors *Actors, cfg *SimulatorCfg) (api.Updateable, error) {

	log := util.NewLogger("Sim")
	sim := &Simulator{
		cfg:           cfg,
		log:           log,
		simsByType:    make(map[api.SimType][]api.Sim),
		simsByName:    make(map[string]api.Sim),
		lastProgCycle: time.Time{},
	}

	// simulator deactivate: don't do additional initialization
	if !sim.cfg.Active {
		return sim, nil
	}

	// extract sims from list of actors
	if err := sim.ExtractSims(actors); err != nil {
		return nil, err
	}
	// initialize sims
	if err := sim.InitSims(); err != nil {
		return nil, err
	}

	return sim, nil
}

// ExtractSims parses all actors and creates references to all sims into the simulator datastructures
func (sim *Simulator) ExtractSims(actors *Actors) error {
	for name, meter := range actors.Meters {
		if simMeter, ok := meter.(api.Sim); ok {
			if err := sim.AssignSim(name, simMeter); err != nil {
				return err
			}
		} else {
			// not a sim - do nothing
			sim.log.DEBUG.Printf("No sim: Meter: %s", name)
		}
	}
	for name, charger := range actors.Chargers {
		if simCharger, ok := charger.(api.Sim); ok {
			if err := sim.AssignSim(name, simCharger); err != nil {
				return err
			}
		} else {
			// not a sim - do nothing
			sim.log.DEBUG.Printf("No sim: Charger: %s", name)
		}
	}
	for name, vehicle := range actors.Vehicles {
		if simVehicle, ok := vehicle.(api.Sim); ok {
			if err := sim.AssignSim(name, simVehicle); err != nil {
				return err
			}
		} else {
			// not a sim - do nothing
			sim.log.DEBUG.Printf("No sim: Vehicle: %s", name)
		}
	}
	for name, switch1p3p := range actors.Switches1p3p {
		if simSwitch, ok := switch1p3p.(api.Sim); ok {
			if err := sim.AssignSim(name, simSwitch); err != nil {
				return err
			}
		} else {
			// not a sim - do nothing
			sim.log.DEBUG.Printf("No sim: Switch1p3p: %s", name)
		}
	}
	return nil
}

// AssignSim assigns the sim by type into the simsByType data structure
// and adds the sim into the simsByName data structure
func (sim *Simulator) AssignSim(name string, simMember api.Sim) error {

	if simType, err := simMember.SimType(); err == nil {
		// check for validity
		switch simType {
		case api.Sim_grid:
			if len(sim.simsByType[api.Sim_grid]) != 0 {
				return fmt.Errorf("only one grid type allowed: %s", name)
			}
		case api.Sim_home:
			if len(sim.simsByType[api.Sim_home]) != 0 {
				return fmt.Errorf("only one home type allowed: %s", name)
			}
		case api.Sim_battery:
			// nothing to do
		case api.Sim_pv:
			// nothing to do
		case api.Sim_switch1p3p:
			// nothing to do
		case api.Hil_switch1p3p:
			// nothing to do
		case api.Sim_vehicle:
			// nothing to do
		case api.Sim_charger:
			// nothing to do

		default:
			return fmt.Errorf("sim [%s]: invalid simType: %d", name, simType)
		}
		// set sim by type
		sim.simsByType[simType] = append(sim.simsByType[simType], simMember)
		// set sims by Name
		sim.simsByName[name] = simMember
		// tell the sim its name (doesn't get it from the config)
		if err := simMember.SetName(name); err != nil {
			return err
		}
		sim.log.DEBUG.Printf("Registered sim: %s", name)
	} else {
		return err
	}
	return nil
}

// InitSims executes the necessary initialization steps for the sims
func (sim *Simulator) InitSims() error {

	// read switch1p3p name from charger and assign switch1p3p reference to charger
	for _, element := range sim.simsByType[api.Sim_charger] {
		if charger, ok := element.(api.SimCharger); ok {
			switch1p3pName, err := charger.Switch1p3pName()
			if err != nil {
				return err
			}
			if switch1p3pName != "" {
				switch1p3p, ok := sim.simsByName[switch1p3pName].(api.SimChargePhases)
				if !ok {
					return fmt.Errorf("no sim found for switch1p3p: %s", switch1p3pName)
				}
				if err := charger.SetSwitch1p3p(switch1p3p); err != nil {
					return err
				}
			}
		} else {
			name, err := element.Name()
			if err != nil {
				return err
			}
			return fmt.Errorf("invalid charger - doesn't fulfill the SimCharger api: %s", name)
		}
	}
	return nil
}

// Update calculates the power distribution and the batteries
func (sim *Simulator) Update() error {
	if !sim.cfg.Active {
		return nil
	}

	if sim.lastProgCycle.IsZero() {
		sim.lastProgCycle = time.Now()
		sim.log.DEBUG.Println("***sim start***")
	} else {
		timeDiff := time.Since(sim.lastProgCycle)
		if sim.progCycle == 0 {
			sim.progCycle = timeDiff
		} else {
			sim.progCycle = (sim.progCycle + timeDiff) / 2
		}
		sim.lastProgCycle = time.Now()
		sim.log.DEBUG.Printf("***sim [%s]***", sim.progCycle.Round(time.Millisecond))

	}

	// execute program steps until end of program or timed program step is reached
	var proceedToNextProgStep bool = true
	for proceedToNextProgStep {
		var err error = nil
		if proceedToNextProgStep, err = sim.ProcessProgramStep(); err != nil {
			return err
		}
	}

	// execute simulator program (checks/changes properties of sims)

	// recalculate everything
	pvPowerW := 0.0
	// first sum the pv generation (>0)
	for _, pv := range sim.simsByType[api.Sim_pv] {
		if power, err := pv.(api.SimMeter).CurrentPower(); err == nil {
			pvPowerW += power
		} else {
			return err
		}
	}

	// sum up consumers ( <0)
	homePowerW := 0.0
	for _, home := range sim.simsByType[api.Sim_home] {
		if power, err := home.(api.SimMeter).CurrentPower(); err == nil {
			homePowerW += power
		} else {
			return err
		}
	}

	// charge power has positive sign when the vehicle is charged
	chargerPowerW := 0.0
	for _, charger := range sim.simsByType[api.Sim_charger] {
		if powerW, err := charger.(api.SimCharger).Update(); err == nil {
			chargerPowerW += powerW
		} else {
			return err
		}
	}

	// handle battery charging / discharging based on the available grid power
	// gridPower > 0: Grid is producer (power would be taken from the grid)
	// gridPower < 0: Grid is consumer (power would be exported to the grid)
	gridPowerW := -(pvPowerW + homePowerW - chargerPowerW)
	if err := sim.updateBatteries(gridPowerW); err != nil {
		return err
	}

	// sum up battery (either consumer or producer)
	batteryPowerW := 0.0
	for _, battery := range sim.simsByType[api.Sim_battery] {
		if power, err := battery.(api.SimBattery).CurrentPower(); err == nil {
			batteryPowerW += power
		} else {
			return err
		}
	}

	// re-calculate the grid power including the batteris
	// < 0 feed into the grid, > 0 get from the grid
	gridPowerW = -(pvPowerW + homePowerW - chargerPowerW + batteryPowerW)
	if err := sim.simsByType[api.Sim_grid][0].(api.SimMeter).SetCurrentPower(gridPowerW); err != nil {
		return err
	}
	sim.log.DEBUG.Println("*********")
	return nil
}

// UpdateBattery checks if batteries can be charged or discharged
// if the gridPower > 0 (generate) power should be taken from batteries (discharge)
// if the gridPower < 0 (consume) batteries can be charged
func (sim *Simulator) updateBatteries(gridPower float64) error {

	// very simple simulation charge first battery with full power, remaining power to the next battery
	// negative grid power would be exported -> we can charge the battery
	availablePower := -gridPower
	for _, simElement := range sim.simsByType[api.Sim_battery] {
		name, err := simElement.Name()
		if err != nil {
			return err
		}
		battery, ok := simElement.(api.SimBattery)
		if !ok {
			return fmt.Errorf("unable to access %s as battery", name)
		}
		chargeDischargePowerW, err := battery.UpdateSoC(availablePower)
		if err != nil {
			return err
		}
		availablePower -= chargeDischargePowerW
	}
	return nil
}

// ProcessProgramStep executes one program step in the simulation file
// returns true when the step is finished (and ready for the next step)
// returns false when the step is in progress or the program is finisehd, or an error occurred
func (sim *Simulator) ProcessProgramStep() (bool, error) {

	if sim.progStep >= len(sim.cfg.Program) {
		// last command already processed. program finished, nothing to do -> return without error
		return false, nil
	}
	executeStep := sim.progStep

	command := sim.cfg.Program[sim.progStep]
	sim.log.DEBUG.Printf("TestCommand: [%d] %v", sim.progStep, command)

	switch command.Cmd {
	case "sleep":
		sleepTime, err := time.ParseDuration(command.Value)
		if err != nil {
			return false, fmt.Errorf("invalid time format in command[%d]: %s", sim.progStep, command.Value)
		}
		if sim.progStepStart.IsZero() {
			sim.progStepStart = time.Now()
			sim.log.DEBUG.Printf("sleep remaining:%v", sleepTime)
		} else {
			// take control loop jitter into account +/- 500ms
			// finish sleep if time is elapsed until the next update call
			if time.Since(sim.progStepStart) >= sleepTime-sim.progCycle-(1*time.Second) {
				// time elapsed, reset start time and go to next step
				// time.Time{} creates a time literal with Go's Zero date
				sim.progStepStart = time.Time{}
				sim.progStep++
				sim.log.DEBUG.Printf("sleep finished with next cycle")
			} else {
				sim.log.DEBUG.Printf("sleep remaining:%v", (sleepTime - time.Since(sim.progStepStart).Round(time.Second)))
			}
		}
	case "set":
		simElement, ok := sim.simsByName[command.Object]
		if !ok {
			return false, fmt.Errorf("unknown object name in command[%d]: [%s]", sim.progStep, command.Object)
		}
		switch command.Attribute {
		case "power":
			value, err := strconv.ParseFloat(command.Value, 64)
			if err != nil {
				return false, err
			}
			if err := simElement.(api.SimPower).SetCurrentPower(value); err != nil {
				return false, err
			}
		case "phases":
			value, err := strconv.ParseInt(command.Value, 10, 32)
			if err != nil {
				return false, err
			}
			// remember old "enabled" status
			enabled, err := simElement.(api.ChargeEnable).Enabled()
			if err != nil {
				return false, err
			}
			if err := simElement.(api.ChargeEnable).Enable(false); err != nil {
				return false, err
			}
			if err := simElement.(api.ChargePhases).Phases1p3p(int(value)); err != nil {
				return false, err
			}
			// switch back to old "enabled" status
			if err := simElement.(api.ChargeEnable).Enable(enabled); err != nil {
				return false, err
			}
		case "lockPhases":
			// value holds the phases value to lock into
			value, err := strconv.ParseInt(command.Value, 10, 32)
			if err != nil {
				return false, err
			}
			if err := simElement.(api.LockPhases1p3p).LockPhases1p3p(int(value)); err != nil {
				return false, err
			}
		case "unlockPhases":
			if err := simElement.(api.LockPhases1p3p).UnlockPhases1p3p(); err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("unknown attribute name in command[%d]: [%s]", sim.progStep, command.Attribute)
		}
		sim.progStep++
	case "setGui":
		switch command.Attribute {
		case "mode":
			var mode api.ChargeMode = api.ModeEmpty
			switch command.Value {
			case api.ModeOff.String():
				mode = api.ModeOff
			case "0": // special situation: "off" in yaml is converted to "0"
				mode = api.ModeOff
			case api.ModeNow.String():
				mode = api.ModeNow
			case api.ModeMinPV.String():
				mode = api.ModeMinPV
			case api.ModePV.String():
				mode = api.ModePV
			default:
				return false, fmt.Errorf("setGui invalid mode value:%s", command.Value)
			}
			if err := sim.SetChargeMode(mode); err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("unknown attribute name in command[%d]: [%s]", sim.progStep, command.Attribute)
		}
		sim.progStep++
	case "expect":
		simElement, ok := sim.simsByName[command.Object]
		if !ok {
			return false, fmt.Errorf("unknown object name in command[%d]: [%s]", sim.progStep, command.Object)
		}
		timeout, err := time.ParseDuration(command.Timeout)
		if err != nil {
			return false, err
		}

		if sim.progStepStart.IsZero() {
			sim.progStepStart = time.Now()
		}
		if time.Since(sim.progStepStart) > timeout {
			// timeout time elapsed, reset start time and return error
			// time.Time{} creates a time literal with Go's Zero date
			sim.progStepStart = time.Time{}
			currentProcStep := sim.progStep
			sim.progStep = len(sim.cfg.Program)
			return false, fmt.Errorf("expect timeout elapsed for command[%d]. Stopping simulator program", currentProcStep)
		} else {
			sim.log.DEBUG.Printf("expect timeout remaining:%v", (timeout - time.Since(sim.progStepStart)))
		}
		switch command.Attribute {
		case "power":
			powerW, err := simElement.(api.Meter).CurrentPower()
			if err != nil {
				return false, err
			}
			expectedValue, err := strconv.ParseFloat(command.Value, 64)
			if err != nil {
				return false, err
			}
			if powerW == expectedValue {
				sim.log.DEBUG.Printf("Expected power reached: %f", expectedValue)
				//Test step OK - switch to next program step
				sim.progStepStart = time.Time{}
				sim.progStep++
			} else {
				sim.log.DEBUG.Printf("Power: %f", powerW)
			}
		case "phases":
			value, err := strconv.ParseInt(command.Value, 10, 32)
			if err != nil {
				return false, err
			}
			phases, err := simElement.(api.SimChargePhases).GetPhases1p3p()
			if err != nil {
				return false, err
			}
			if int64(phases) == value {
				sim.log.DEBUG.Printf("Expected phases reached: %d", value)
				sim.progStepStart = time.Time{}
				sim.progStep++
			} else {
				sim.log.DEBUG.Printf("Phases: %d", phases)
			}
		default:
			return false, fmt.Errorf("unknown attribute name in command[%d]: [%s]", sim.progStep, command.Attribute)
		}
	case "connect":
		simCharger, ok := sim.simsByName[command.Object].(api.SimCharger)
		if !ok {
			return false, fmt.Errorf("unknown object name in command[%d]: [%s]", sim.progStep, command.Object)
		}
		simVehicle, ok := sim.simsByName[command.Value].(api.SimVehicle)
		if !ok {
			return false, fmt.Errorf("unknown vehicle name in command[%d]: [%s]", sim.progStep, command.Value)
		}
		if err := simCharger.Connect(simVehicle); err != nil {
			return false, err
		}
		sim.progStep++
	case "disconnect":
		simCharger, ok := sim.simsByName[command.Object].(api.SimCharger)
		if !ok {
			return false, fmt.Errorf("unknown object name in command[%d]: [%s]", sim.progStep, command.Object)
		}
		if err := simCharger.Disconnect(); err != nil {
			return false, err
		}
		sim.progStep++
	default:
		return false, fmt.Errorf("unhandled command: [%s]", command.Cmd)
	}

	if sim.progStep >= len(sim.cfg.Program) {
		sim.log.DEBUG.Printf("*** Program finished ***")
		return false, nil
	}

	if executeStep != sim.progStep {
		// program step finished
		return true, nil
	}
	return false, nil
}

// SetChargeMode sets the charge mode using the Rest API
// TODO: get url from evcc.yaml
func (sim *Simulator) SetChargeMode(mode api.ChargeMode) error {

	// define the requested mode
	address := "http://0.0.0.0:7070"
	url := address + "/api/loadpoints/0/mode/" + mode.String()

	// send the requested mode
	resp, err := http.Post(url, "application/json; charset=utf-8", nil)
	if err != nil {
		return err
	}

	// the post request returns a json result with the active mode
	// example: {"result":"now"}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := struct {
		Result string
	}{
		Result: "",
	}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return err
	}
	// check if the requested mode was set
	if data.Result != mode.String() {
		return fmt.Errorf("failed to set the requested mode: %v. Result mode:%s", mode, data.Result)
	}
	return nil
}
