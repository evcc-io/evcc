package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/fatih/structs"
)

type dumper struct {
	len int
}

func (d *dumper) Header(name, underline string) {
	fmt.Println(name)
	fmt.Println(strings.Repeat(underline, len(name)))
}

func (d *dumper) DumpWithHeader(name string, device interface{}) {
	if d.len > 1 {
		d.Header(name, "-")
	}

	d.Dump(name, device)

	if d.len > 1 {
		fmt.Println()
	}
}

// bo returns an exponential backoff for reading meter power quickly
func bo() *backoff.ExponentialBackOff {
	return backoff.NewExponentialBackOff(backoff.WithInitialInterval(20*time.Millisecond), backoff.WithMaxElapsedTime(time.Second))
}

func (d *dumper) Dump(name string, v interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	var isHeating bool
	if fd, ok := v.(api.FeatureDescriber); ok {
		isHeating = slices.Contains(fd.Features(), api.Heating)
	}

	// Start overall timing
	totalStart := time.Now()

	// meter

	if v, ok := v.(api.Meter); ok {
		start := time.Now()
		power, err := backoff.RetryWithData(func() (float64, error) {
			f, err := v.CurrentPower()
			if err != nil {
				fmt.Println(err)
			}
			return f, err
		}, bo())
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Power:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Power:\t%.0fW (%v)\n", power, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.MeterEnergy); ok {
		start := time.Now()
		energy, err := v.TotalEnergy()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Energy:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Energy:\t%.1fkWh (%v)\n", energy, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.PhaseCurrents); ok {
		start := time.Now()
		i1, i2, i3, err := v.Currents()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Current L1..L3:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Current L1..L3:\t%.3gA %.3gA %.3gA (%v)\n", i1, i2, i3, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.PhaseVoltages); ok {
		start := time.Now()
		u1, u2, u3, err := v.Voltages()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Voltage L1..L3:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Voltage L1..L3:\t%.3gV %.3gV %.3gV (%v)\n", u1, u2, u3, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.PhasePowers); ok {
		start := time.Now()
		p1, p2, p3, err := v.Powers()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Power L1..L3:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Power L1..L3:\t%.0fW %.0fW %.0fW (%v)\n", p1, p2, p3, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.Battery); ok {
		var soc float64
		var err error

		// wait up to 1m for the vehicle to wakeup
		start := time.Now()
		for err = api.ErrMustRetry; err != nil && errors.Is(err, api.ErrMustRetry); {
			if soc, err = v.Soc(); err != nil {
				if time.Since(start) > time.Minute {
					err = os.ErrDeadlineExceeded
				} else {
					fmt.Fprint(w, ".")
					time.Sleep(3 * time.Second)
				}
			}
		}
		duration := time.Since(start)

		if isHeating {
			if err != nil {
				fmt.Fprintf(w, "Temp:\t%v\n", err)
			} else {
				fmt.Fprintf(w, "Temp:\t%.0f°C (%v)\n", soc, duration.Round(time.Millisecond))
			}
		} else {
			if err != nil {
				fmt.Fprintf(w, "Soc:\t%v\n", err)
			} else {
				fmt.Fprintf(w, "Soc:\t%.0f%% (%v)\n", soc, duration.Round(time.Millisecond))
			}
		}
	}

	if v, ok := v.(api.BatteryCapacity); ok {
		fmt.Fprintf(w, "Capacity:\t%.1fkWh\n", v.Capacity())
	}

	if v, ok := v.(api.MaxACPowerGetter); ok {
		fmt.Fprintf(w, "Max AC power:\t%.0fW\n", v.MaxACPower())
	}

	// charger

	if v, ok := v.(api.ChargeState); ok {
		start := time.Now()
		status, err := v.Status()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Charge status:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Charge status:\t%v (%v)\n", status, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.StatusReasoner); ok {
		start := time.Now()
		status, err := v.StatusReason()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Status reason:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Status reason:\t%v (%v)\n", status, duration.Round(time.Millisecond))
		}
	}

	// controllable battery
	if _, ok := v.(api.BatteryController); ok {
		fmt.Fprintf(w, "Controllable:\ttrue\n")
	}

	if v, ok := v.(api.Charger); ok {
		start := time.Now()
		enabled, err := v.Enabled()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Enabled:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Enabled:\t%t (%v)\n", enabled, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.ChargeRater); ok {
		start := time.Now()
		energy, err := v.ChargedEnergy()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Charged:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Charged:\t%.1fkWh (%v)\n", energy, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.ChargeTimer); ok {
		start := time.Now()
		chargeDuration, err := v.ChargeDuration()
		queryDuration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Duration:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Duration:\t%v (%v)\n", chargeDuration.Truncate(time.Second), queryDuration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.CurrentLimiter); ok {
		start := time.Now()
		min, max, err := v.GetMinMaxCurrent()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Mix/Max Current:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Mix/Max Current:\t%.1f/%.1fA (%v)\n", min, max, duration.Round(time.Millisecond))
		}
	}

	// vehicle

	if v, ok := v.(api.VehicleRange); ok {
		start := time.Now()
		rng, err := v.Range()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Range:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Range:\t%vkm (%v)\n", rng, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.VehicleOdometer); ok {
		start := time.Now()
		odo, err := v.Odometer()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Odometer:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Odometer:\t%.0fkm (%v)\n", odo, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.VehicleFinishTimer); ok {
		start := time.Now()
		ft, err := v.FinishTime()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Finish time:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Finish time:\t%v (%v)\n", ft.Truncate(time.Minute).In(time.Local), duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.VehicleClimater); ok {
		start := time.Now()
		active, err := v.Climater()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Climater:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Climate active:\t%v (%v)\n", active, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.VehiclePosition); ok {
		start := time.Now()
		lat, lon, err := v.Position()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Position:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Position:\t%v,%v (%v)\n", lat, lon, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.SocLimiter); ok {
		start := time.Now()
		limitSoc, err := v.GetLimitSoc()
		duration := time.Since(start)

		if isHeating {
			if err != nil {
				fmt.Fprintf(w, "Max Temp:\t%v\n", err)
			} else {
				fmt.Fprintf(w, "Max Temp:\t%d°C (%v)\n", limitSoc, duration.Round(time.Millisecond))
			}
		} else {
			if err != nil {
				fmt.Fprintf(w, "Limit Soc:\t%v\n", err)
			} else {
				fmt.Fprintf(w, "Limit Soc:\t%d%% (%v)\n", limitSoc, duration.Round(time.Millisecond))
			}
		}
	}

	if v, ok := v.(api.Vehicle); ok {
		if len(v.Identifiers()) > 0 {
			fmt.Fprintf(w, "Identifiers:\t%v\n", v.Identifiers())
		}
		if !structs.IsZero(v.OnIdentified()) {
			fmt.Fprintf(w, "OnIdentified:\t%s\n", v.OnIdentified())
		}
	}

	// currents and phases

	if v, ok := v.(api.CurrentGetter); ok {
		start := time.Now()
		f, err := v.GetMaxCurrent()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Max Current:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Max Current:\t%.1fA (%v)\n", f, duration.Round(time.Millisecond))
		}
	}

	if v, ok := v.(api.PhaseGetter); ok {
		start := time.Now()
		f, err := v.GetPhases()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Phases:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Phases:\t%d (%v)\n", f, duration.Round(time.Millisecond))
		}
	}

	// Identity

	if v, ok := v.(api.Identifier); ok {
		start := time.Now()
		id, err := v.Identify()
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(w, "Identifier:\t%v\n", err)
		} else {
			if id == "" {
				id = "<none>"
			}
			fmt.Fprintf(w, "Identifier:\t%s (%v)\n", id, duration.Round(time.Millisecond))
		}
	}

	// features

	if v, ok := v.(api.FeatureDescriber); ok {
		if ff := v.Features(); len(ff) > 0 {
			fmt.Fprintf(w, "Features:\t%v\n", ff)
		}
	}

	w.Flush()

	// Output total query time
	totalDuration := time.Since(totalStart)
	fmt.Printf("\nTotal query time: %v\n", totalDuration.Round(time.Millisecond))
}

func (d *dumper) DumpDiagnosis(v interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	if v, ok := v.(api.Diagnosis); ok {
		fmt.Fprintln(w, "Diagnostic dump:")
		v.Diagnose()
	}

	w.Flush()
}
