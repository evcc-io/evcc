package cmd

import (
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

// meterVerifyBatteryModesCmd represents the meter verifybatterymodes command
var meterVerifyBatteryModesCmd = &cobra.Command{
	Use:   "verifybatterymodes [name]",
	Short: "Cycle through all battery mode transitions and verify battery power interactively",
	Args:  cobra.MaximumNArgs(1),
	Run:   runMeterVerifyBatteryModes,
}

func init() {
	meterCmd.AddCommand(meterVerifyBatteryModesCmd)
	withCustomTemplate(meterVerifyBatteryModesCmd)

	meterVerifyBatteryModesCmd.Flags().Duration(flagDelay, 30*time.Second, "observation window after setting mode (battery power is polled every second)")
}

// batteryModeExpectation documents the expected battery power sign per mode (negative: charging, positive: discharging)
var batteryModeExpectation = map[api.BatteryMode]string{
	api.BatteryNormal:     "any sign (< 0 with pv surplus, > 0 with house demand)",
	api.BatteryHold:       "<= 0 (no discharging, charging from pv surplus allowed)",
	api.BatteryCharge:     "< 0 (charging from grid)",
	api.BatteryHoldCharge: ">= 0 (no charging, discharging allowed)",
	api.BatteryDischarge:  "> 0 (discharging to grid)",
}

// batteryModeTransitions returns the sequence of target modes that, starting from modes[0],
// visits every ordered pair of modes exactly once and ends at modes[0]
func batteryModeTransitions(modes []api.BatteryMode) []api.BatteryMode {
	n := len(modes)
	var res []api.BatteryMode

	for i := 0; i < n-1; i++ {
		for j := n - 1; j > i+1; j-- {
			res = append(res, modes[j], modes[i])
		}
		res = append(res, modes[i+1])
	}

	for k := n - 2; k >= 0; k-- {
		res = append(res, modes[k])
	}

	return res
}

// batteryPowerState labels the battery power sign
func batteryPowerState(p float64) string {
	switch {
	case p < 0:
		return "charging"
	case p > 0:
		return "discharging"
	default:
		return "idle"
	}
}

// observeBatteryPower polls and prints battery power once per second for the given duration and returns the last reading
func observeBatteryPower(m api.Meter, d time.Duration) string {
	var last string
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(time.Second) {
		if p, err := m.CurrentPower(); err != nil {
			last = err.Error()
		} else {
			last = fmt.Sprintf("%.0fW (%s)", p, batteryPowerState(p))
		}
		fmt.Printf("\rbattery power: %-40s", last)
	}
	fmt.Println()

	return last
}

func verifyBatteryModes(m api.Meter, bc api.BatteryController, modes []api.BatteryMode, delay time.Duration) {
	setMode := func(mode api.BatteryMode) {
		if err := bc.SetBatteryMode(mode); err != nil {
			log.FATAL.Fatalln("set battery mode:", err)
		}
	}

	fmt.Printf("modes: %v\ntransitions: %d\ndelay: %s\n", modes, len(modes)*(len(modes)-1), delay)

	from := modes[0]
	fmt.Printf("\ninitial mode: %s\n", from)
	setMode(from)

	var protocol []string
	for _, to := range batteryModeTransitions(modes) {
		fmt.Printf("\n%s -> %s\nexpected battery power: %s\n", from, to, batteryModeExpectation[to])
		setMode(to)

		power := observeBatteryPower(m, delay)

		var ok bool
		if err := survey.AskOne(&survey.Confirm{Message: "Battery power as expected?", Default: true}, &ok); err != nil {
			log.FATAL.Fatal(err)
		}

		protocol = append(protocol, fmt.Sprintf("%-10s -> %-10s %-24s %s", from, to, power, lo.Ternary(ok, "ok", "FAILED")))
		from = to
	}

	fmt.Println("\nprotocol:")
	for _, line := range protocol {
		fmt.Println(line)
	}
}

func runMeterVerifyBatteryModes(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, &conf); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := configureMeters(conf.Meters, args...); err != nil {
		log.FATAL.Fatal(err)
	}

	delay, _ := cmd.Flags().GetDuration(flagDelay)

	for _, dev := range config.Meters().Devices() {
		v := dev.Instance()

		bc, ok := api.Cap[api.BatteryController](v)
		if !ok {
			continue
		}

		modes := lo.Without(bc.BatteryModes(), api.BatteryUnknown)
		if len(modes) < 2 {
			log.WARN.Printf("%s: not enough battery modes to verify transitions: %v", dev.Config().Name, modes)
			continue
		}

		fmt.Println(deviceHeader(dev))
		verifyBatteryModes(v, bc, modes, delay)
	}
}
