package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

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

func (d *dumper) Dump(name string, v interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	// meter

	if v, ok := v.(api.Meter); ok {
		if power, err := v.CurrentPower(); err != nil {
			fmt.Fprintf(w, "Power:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Power:\t%.0fW\n", power)
		}
	}

	if v, ok := v.(api.MeterEnergy); ok {
		if energy, err := v.TotalEnergy(); err != nil {
			fmt.Fprintf(w, "Energy:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Energy:\t%.1fkWh\n", energy)
		}
	}

	if v, ok := v.(api.PhaseCurrents); ok {
		if i1, i2, i3, err := v.Currents(); err != nil {
			fmt.Fprintf(w, "Current L1..L3:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Current L1..L3:\t%.3gA %.3gA %.3gA\n", i1, i2, i3)
		}
	}

	if v, ok := v.(api.PhaseVoltages); ok {
		if u1, u2, u3, err := v.Voltages(); err != nil {
			fmt.Fprintf(w, "Voltage L1..L3:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Voltage L1..L3:\t%.3gV %.3gV %.3gV\n", u1, u2, u3)
		}
	}

	if v, ok := v.(api.PhasePowers); ok {
		if p1, p2, p3, err := v.Powers(); err != nil {
			fmt.Fprintf(w, "Power L1..L3:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Power L1..L3:\t%.3gW %.3gW %.3gW\n", p1, p2, p3)
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

		if err != nil {
			fmt.Fprintf(w, "Soc:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Soc:\t%.0f%%\n", soc)
		}
	}

	if v, ok := v.(api.BatteryCapacity); ok {
		fmt.Fprintf(w, "Capacity:\t%.1fkWh\n", v.Capacity())
	}

	// charger

	if v, ok := v.(api.ChargeState); ok {
		if status, err := v.Status(); err != nil {
			fmt.Fprintf(w, "Charge status:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Charge status:\t%v\n", status)
		}
	}

	if v, ok := v.(api.Charger); ok {
		if enabled, err := v.Enabled(); err != nil {
			fmt.Fprintf(w, "Enabled:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Enabled:\t%t\n", enabled)
		}
	}

	if v, ok := v.(api.ChargeRater); ok {
		if energy, err := v.ChargedEnergy(); err != nil {
			fmt.Fprintf(w, "Charged:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Charged:\t%.1fkWh\n", energy)
		}
	}

	if v, ok := v.(api.ChargeTimer); ok {
		if duration, err := v.ChargingTime(); err != nil {
			fmt.Fprintf(w, "Duration:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Duration:\t%v\n", duration.Truncate(time.Second))
		}
	}

	// vehicle

	if v, ok := v.(api.VehicleRange); ok {
		if rng, err := v.Range(); err != nil {
			fmt.Fprintf(w, "Range:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Range:\t%vkm\n", rng)
		}
	}

	if v, ok := v.(api.VehicleOdometer); ok {
		if odo, err := v.Odometer(); err != nil {
			fmt.Fprintf(w, "Odometer:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Odometer:\t%.0fkm\n", odo)
		}
	}

	if v, ok := v.(api.VehicleFinishTimer); ok {
		if ft, err := v.FinishTime(); err != nil {
			fmt.Fprintf(w, "Finish time:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Finish time:\t%v\n", ft.Truncate(time.Minute).In(time.Local))
		}
	}

	if v, ok := v.(api.VehicleClimater); ok {
		if active, err := v.Climater(); err != nil {
			fmt.Fprintf(w, "Climater:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Climate active:\t%v\n", active)
		}
	}

	if v, ok := v.(api.VehiclePosition); ok {
		if lat, lon, err := v.Position(); err != nil {
			fmt.Fprintf(w, "Position:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Position:\t%v,%v\n", lat, lon)
		}
	}

	if v, ok := v.(api.SocLimiter); ok {
		if targetSoc, err := v.TargetSoc(); err != nil {
			fmt.Fprintf(w, "Target Soc:\t%v\n", err)
		} else {
			fmt.Fprintf(w, "Target Soc:\t%.0f%%\n", targetSoc)
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

	// Identity

	if v, ok := v.(api.Identifier); ok {
		if id, err := v.Identify(); err != nil {
			fmt.Fprintf(w, "Identifier:\t%v\n", err)
		} else {
			if id == "" {
				id = "<none>"
			}
			fmt.Fprintf(w, "Identifier:\t%s\n", id)
		}
	}

	w.Flush()
}

func (d *dumper) DumpDiagnosis(v interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	if v, ok := v.(api.Diagnosis); ok {
		fmt.Fprintln(w, "Diagnostic dump:")
		v.Diagnose()
	}

	w.Flush()
}
