package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	sunspec "github.com/andig/gosunspec"
	bus "github.com/andig/gosunspec/modbus"
	"github.com/andig/gosunspec/smdx"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volkszaehler/mbmd/meters"
	quirks "github.com/volkszaehler/mbmd/meters/sunspec"
)

// sunspecCmd represents the charger command
var sunspecCmd = &cobra.Command{
	Use:   "sunspec <connection>",
	Short: "Dump SunSpec model information",
	Args:  cobra.ExactArgs(1),
	Run:   runSunspec,
}

var slaveID *int

func init() {
	rootCmd.AddCommand(sunspecCmd)

	slaveID = sunspecCmd.Flags().IntP("id", "i", 1, "Slave id")
}

func pf(format string, v ...interface{}) {
	format = strings.TrimSuffix(format, "\n") + "\n"
	fmt.Printf(format, v...)
}

func modelName(m sunspec.Model) string {
	model := smdx.GetModel(uint16(m.Id()))
	if model == nil {
		return ""
	}
	return model.Name
}

func runSunspec(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), nil)

	conn := meters.NewTCP(args[0])
	conn.Slave(uint8(*slaveID))
	conn.Logger(log.TRACE)

	in, err := bus.Open(conn.ModbusClient())
	if err != nil && in == nil {
		log.FATAL.Fatal(err)
	} else if err != nil {
		log.WARN.Printf("warning: device opened with partial result: %v", err) // log error but continue
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	in.Do(func(d sunspec.Device) {
		d.Do(func(m sunspec.Model) {
			pf("--------- Model %d %s ---------", m.Id(), modelName(m))

			blocknum := 0
			m.Do(func(b sunspec.Block) {
				if blocknum > 0 {
					fmt.Fprintf(tw, "-- Block %d --\n", blocknum)
				}
				blocknum++

				err = b.Read()
				if err != nil {
					log.INFO.Printf("skipping due to read error: %v", err)
					return
				}

				b.Do(func(p sunspec.Point) {
					t := p.Type()[0:3]
					v := p.Value()
					if p.NotImplemented() {
						v = "n/a"
					} else if t == "int" || t == "uin" || t == "acc" {
						// for time being, always to this
						quirks.FixKostal(p)

						v = p.ScaledValue()
						v = fmt.Sprintf("%.2f", v)
					}

					vs := fmt.Sprintf("%17v", v)
					fmt.Fprintf(tw, "%s\t%s\t   %s\n", p.Id(), vs, p.Type())
				})
			})

			tw.Flush()
		})
	})
}
