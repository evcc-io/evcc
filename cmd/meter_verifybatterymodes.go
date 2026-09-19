package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/olekukonko/tablewriter"
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
	if len(modes) < 2 {
		return nil
	}

	var res []api.BatteryMode

	for i := range len(modes) - 1 {
		for _, m := range modes[i+2:] {
			res = append(res, m, modes[i])
		}
		res = append(res, modes[i+1])
	}

	for _, m := range slices.Backward(modes[:len(modes)-1]) {
		res = append(res, m)
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

// observeBatteryPower polls and prints battery power once per second for the given duration.
// Returns the last reading and whether it was a successful measurement.
func observeBatteryPower(m api.Meter, d time.Duration) (string, bool) {
	var last string
	var ok bool
	for range max(1, int(d/time.Second)) {
		p, err := m.CurrentPower()
		if ok = err == nil; ok {
			last = fmt.Sprintf("%.0fW (%s)", p, batteryPowerState(p))
		} else {
			last = err.Error()
		}
		fmt.Printf("\rbattery power: %-40s", last)
		time.Sleep(time.Second)
	}
	fmt.Println()

	return last, ok
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

	var protocol [][]string
	for _, to := range batteryModeTransitions(modes) {
		fmt.Printf("\n%s -> %s\nexpected battery power: %s\n", from, to, batteryModeExpectation[to])
		setMode(to)

		power, ok := observeBatteryPower(m, delay)

		// a failed measurement defaults to not ok
		if err := survey.AskOne(&survey.Confirm{Message: "Battery power as expected?", Default: ok}, &ok); err != nil {
			log.FATAL.Fatal(err)
		}

		expected, _, _ := strings.Cut(batteryModeExpectation[to], " (")
		protocol = append(protocol, []string{fmt.Sprint(len(protocol) + 1), from.String(), to.String(), expected, power, lo.Ternary(ok, "ok", "FAILED")})
		from = to
	}

	fmt.Println()
	table := tablewriter.NewTable(os.Stdout)
	table.Header([]string{"#", "from", "to", "expected", "observed", "result"})
	for _, row := range protocol {
		table.Append(row)
	}
	table.Render()
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

	batteries := lo.Filter(config.Meters().Devices(), func(dev config.Device[api.Meter], _ int) bool {
		return api.HasCap[api.BatteryController](dev.Instance())
	})

	switch {
	case len(batteries) == 0:
		log.FATAL.Fatal("no meter with battery mode control found")
	case len(args) == 0 && len(batteries) > 1:
		names := lo.Map(batteries, func(dev config.Device[api.Meter], _ int) string { return dev.Config().Name })
		log.FATAL.Fatalf("multiple meters with battery mode control found, specify one: %s", strings.Join(names, ", "))
	}

	dev := batteries[0]
	v := dev.Instance()
	bc, _ := api.Cap[api.BatteryController](v)

	modes := lo.Without(bc.BatteryModes(), api.BatteryUnknown)
	if len(modes) < 2 {
		log.FATAL.Fatalf("%s: not enough battery modes to verify transitions: %v", dev.Config().Name, modes)
	}

	fmt.Println(deviceHeader(dev))
	verifyBatteryModes(v, bc, modes, delay)
}
