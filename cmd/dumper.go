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
	len     int
	timeout time.Duration
}

func (d *dumper) Header(name, underline string) {
	fmt.Println(name)
	fmt.Println(strings.Repeat(underline, len(name)))
}

func (d *dumper) DumpWithHeader(name string, device any) {
	if d.len > 1 {
		d.Header(name, "-")
	}

	d.Dump(name, device)

	if d.len > 1 {
		fmt.Println()
	}
}

// bo returns an exponential backoff for reading meter power quickly
func (d *dumper) bo() *backoff.ExponentialBackOff {
	return backoff.NewExponentialBackOff(backoff.WithInitialInterval(20*time.Millisecond), backoff.WithMaxElapsedTime(d.timeout))
}

// formatDuration returns duration as string if >= 1ms, otherwise empty string
func formatDuration(duration time.Duration) string {
	duration = duration.Round(time.Millisecond)
	if duration >= time.Millisecond {
		return duration.String()
	}
	return ""
}

// measureTime executes a function, measures its duration, and prints the result with timing
func (d *dumper) measureTime(w *tabwriter.Writer, label string, fn func() (string, error)) {
	start := time.Now()
	value, err := fn()

	if err != nil {
		fmt.Fprintf(w, "%s:\t%v\t%s\t\n", label, err, formatDuration(time.Since(start)))
	} else {
		fmt.Fprintf(w, "%s:\t%s\t%s\t\n", label, value, formatDuration(time.Since(start)))
	}
}

func (d *dumper) Dump(name string, v any) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	var isHeating bool
	if fd, ok := api.Cap[api.FeatureDescriber](v); ok {
		isHeating = slices.Contains(fd.Features(), api.Heating)
	}

	// Start overall timing
	totalStart := time.Now()

	// meter

	if v, ok := api.Cap[api.Meter](v); ok {
		d.measureTime(w, "Power", func() (string, error) {
			power, err := backoff.RetryWithData(func() (float64, error) {
				f, err := v.CurrentPower()
				return f, err
			}, d.bo())
			return fmt.Sprintf("%.0fW", power), err
		})
	}

	if v, ok := api.Cap[api.MeterImport](v); ok {
		d.measureTime(w, "Import", func() (string, error) {
			energy, err := v.ImportEnergy()
			return fmt.Sprintf("%.1fkWh", energy), err
		})
	}

	if v, ok := api.Cap[api.MeterExport](v); ok {
		d.measureTime(w, "Export", func() (string, error) {
			energy, err := v.ExportEnergy()
			return fmt.Sprintf("%.1fkWh", energy), err
		})
	}

	if v, ok := api.Cap[api.PhaseCurrents](v); ok {
		d.measureTime(w, "Current L1..L3", func() (string, error) {
			i1, i2, i3, err := v.Currents()
			return fmt.Sprintf("%.3gA %.3gA %.3gA", i1, i2, i3), err
		})
	}

	if v, ok := api.Cap[api.PhaseVoltages](v); ok {
		d.measureTime(w, "Voltage L1..L3", func() (string, error) {
			u1, u2, u3, err := v.Voltages()
			return fmt.Sprintf("%.3gV %.3gV %.3gV", u1, u2, u3), err
		})
	}

	if v, ok := api.Cap[api.PhasePowers](v); ok {
		d.measureTime(w, "Power L1..L3", func() (string, error) {
			p1, p2, p3, err := v.Powers()
			return fmt.Sprintf("%.0fW %.0fW %.0fW", p1, p2, p3), err
		})
	}

	if v, ok := api.Cap[api.Battery](v); ok {
		label := "Soc"
		format := "%.0f%%"
		if isHeating {
			label = "Temp"
			format = "%.0f°C"
		}

		start := time.Now()
		var soc float64
		var err error

		// wait up to 1m for the vehicle to wakeup
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

		if err != nil {
			fmt.Fprintf(w, "%s:\t%v\t%s\t\n", label, err, formatDuration(time.Since(start)))
		} else {
			fmt.Fprintf(w, "%s:\t%s\t%s\t\n", label, fmt.Sprintf(format, soc), formatDuration(time.Since(start)))
		}
	}

	if v, ok := api.Cap[api.BatteryCapacity](v); ok {
		fmt.Fprintf(w, "Capacity:\t%.1fkWh\t\t\n", v.Capacity())
	}

	if v, ok := api.Cap[api.BatterySocLimiter](v); ok {
		min, max := v.GetSocLimits()
		fmt.Fprintf(w, "Min soc:\t%.0f%%\t\t\n", min)
		fmt.Fprintf(w, "Max soc:\t%.0f%%\t\t\n", max)
	}

	if v, ok := api.Cap[api.BatteryPowerLimiter](v); ok {
		charge, discharge := v.GetPowerLimits()
		fmt.Fprintf(w, "Charge power:\t%.0fW\t\t\n", charge)
		fmt.Fprintf(w, "Discharge power:\t%.0fW\t\t\n", discharge)
	}

	if v, ok := api.Cap[api.MaxACPowerGetter](v); ok {
		fmt.Fprintf(w, "Max AC power:\t%.0fW\t\t\n", v.MaxACPower())
	}

	if v, ok := api.Cap[api.Dimmer](v); ok {
		d.measureTime(w, "Dimmed", func() (string, error) {
			dimmed, err := v.Dimmed()
			return fmt.Sprintf("%t", dimmed), err
		})
	}

	if v, ok := api.Cap[api.Curtailer](v); ok {
		d.measureTime(w, "Curtailed", func() (string, error) {
			curtailed, err := v.Curtailed()
			return fmt.Sprintf("%t", curtailed), err
		})
	}

	// charger

	if v, ok := api.Cap[api.ChargeState](v); ok {
		d.measureTime(w, "Charge status", func() (string, error) {
			status, err := v.Status()
			return fmt.Sprintf("%v", status), err
		})
	}

	if v, ok := api.Cap[api.StatusReasoner](v); ok {
		d.measureTime(w, "Status reason", func() (string, error) {
			status, err := v.StatusReason()
			return fmt.Sprintf("%v", status), err
		})
	}

	// controllable battery
	if api.HasCap[api.BatteryController](v) {
		fmt.Fprintf(w, "Controllable:\ttrue\t\t\n")
	}

	if v, ok := api.Cap[api.Charger](v); ok {
		d.measureTime(w, "Enabled", func() (string, error) {
			enabled, err := v.Enabled()
			return fmt.Sprintf("%t", enabled), err
		})
	}

	if v, ok := api.Cap[api.ChargeRater](v); ok {
		d.measureTime(w, "Charged", func() (string, error) {
			energy, err := v.ChargedEnergy()
			return fmt.Sprintf("%.1fkWh", energy), err
		})
	}

	if v, ok := api.Cap[api.ChargeTimer](v); ok {
		d.measureTime(w, "Duration", func() (string, error) {
			chargeDuration, err := v.ChargeDuration()
			return fmt.Sprintf("%v", chargeDuration.Truncate(time.Second)), err
		})
	}

	if v, ok := api.Cap[api.CurrentLimiter](v); ok {
		d.measureTime(w, "Mix/Max Current", func() (string, error) {
			min, max, err := v.GetMinMaxCurrent()
			return fmt.Sprintf("%.1f/%.1fA", min, max), err
		})
	}

	// vehicle

	if v, ok := api.Cap[api.VehicleRange](v); ok {
		d.measureTime(w, "Range", func() (string, error) {
			rng, err := v.Range()
			return fmt.Sprintf("%vkm", rng), err
		})
	}

	if v, ok := api.Cap[api.VehicleOdometer](v); ok {
		d.measureTime(w, "Odometer", func() (string, error) {
			odo, err := v.Odometer()
			return fmt.Sprintf("%.0fkm", odo), err
		})
	}

	if v, ok := api.Cap[api.VehicleFinishTimer](v); ok {
		d.measureTime(w, "Finish time", func() (string, error) {
			ft, err := v.FinishTime()
			return fmt.Sprintf("%v", ft.Truncate(time.Minute).In(time.Local)), err
		})
	}

	if v, ok := api.Cap[api.VehicleClimater](v); ok {
		d.measureTime(w, "Climate active", func() (string, error) {
			active, err := v.Climater()
			return fmt.Sprintf("%v", active), err
		})
	}

	if v, ok := api.Cap[api.VehiclePosition](v); ok {
		d.measureTime(w, "Position", func() (string, error) {
			lat, lon, err := v.Position()
			return fmt.Sprintf("%v,%v", lat, lon), err
		})
	}

	if v, ok := api.Cap[api.SocLimiter](v); ok {
		label := "Limit Soc"
		format := "%d%%"
		if isHeating {
			label = "Max Temp"
			format = "%d°C"
		}
		d.measureTime(w, label, func() (string, error) {
			limitSoc, err := v.GetLimitSoc()
			return fmt.Sprintf(format, limitSoc), err
		})
	}

	if v, ok := api.Cap[api.Vehicle](v); ok {
		if len(v.Identifiers()) > 0 {
			fmt.Fprintf(w, "Identifiers:\t%v\t\t\n", v.Identifiers())
		}
		if !structs.IsZero(v.OnIdentified()) {
			fmt.Fprintf(w, "OnIdentified:\t%s\t\t\n", v.OnIdentified())
		}
	}

	// currents and phases

	if v, ok := api.Cap[api.CurrentGetter](v); ok {
		d.measureTime(w, "Max Current", func() (string, error) {
			f, err := v.GetMaxCurrent()
			return fmt.Sprintf("%.1fA", f), err
		})
	}

	if v, ok := api.Cap[api.PhaseGetter](v); ok {
		d.measureTime(w, "Phases", func() (string, error) {
			f, err := v.GetPhases()
			return fmt.Sprintf("%d", f), err
		})
	}

	// Identity

	if v, ok := api.Cap[api.Identifier](v); ok {
		d.measureTime(w, "Identifier", func() (string, error) {
			id, err := v.Identify()
			if err == nil && id == "" {
				id = "<none>"
			}
			return id, err
		})
	}

	// features

	if v, ok := api.Cap[api.FeatureDescriber](v); ok {
		if ff := v.Features(); len(ff) > 0 {
			fmt.Fprintf(w, "Features:\t%v\t\t\n", ff)
		}
	}

	if totalDurationStr := formatDuration(time.Since(totalStart)); totalDurationStr != "" {
		fmt.Fprintf(w, "\t\t\t\nTotal time:\t\t%s\t\n", totalDurationStr)
	}

	w.Flush()
}

func (d *dumper) DumpDiagnosis(v any) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	if v, ok := api.Cap[api.Diagnosis](v); ok {
		fmt.Fprintln(w, "Diagnostic dump:")
		v.Diagnose()
	}

	w.Flush()
}
