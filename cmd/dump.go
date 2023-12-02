package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
)

// dumpCmd represents the meter command
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump configuration",
	Run:   runDump,
}

var (
	//go:embed dump.tpl
	dumpTmpl string

	dumpConfig *bool
)

func init() {
	rootCmd.AddCommand(dumpCmd)

	dumpConfig = dumpCmd.Flags().Bool("cfg", false, "Dump config file")
}

func handle[T any](name string, h config.Handler[T]) config.Device[T] {
	dev, err := h.ByName(name)
	if err != nil {
		log.FATAL.Fatal(err)
	}
	return dev
}

func runDump(cmd *cobra.Command, args []string) {
	// load config
	err := loadConfigFile(&conf)

	// setup environment
	if err == nil {
		err = configureEnvironment(cmd, conf)
	}

	var site *core.Site
	if err == nil {
		site, err = configureSiteAndLoadpoints(conf)
	}

	if *dumpConfig {
		file, pathErr := filepath.Abs(cfgFile)
		if pathErr != nil {
			file = cfgFile
		}

		var redacted string
		if src, err := os.ReadFile(cfgFile); err == nil {
			redacted = redact(string(src))
		}

		tmpl := template.Must(
			template.New("dump").
				Funcs(sprig.TxtFuncMap()).
				Parse(dumpTmpl))

		out := new(bytes.Buffer)
		_ = tmpl.Execute(out, map[string]any{
			"CfgFile":    file,
			"CfgError":   errorString(err),
			"CfgContent": redacted,
			"Version":    server.FormattedVersion(),
		})

		fmt.Println(out.String())

		os.Exit(0)
	}

	if err != nil {
		log.FATAL.Fatal(err)
	}

	d := dumper{len: 2}

	d.Header("config", "=")
	fmt.Println("")

	if name := site.Meters.GridMeterRef; name != "" {
		d.DumpWithHeader(fmt.Sprintf("grid: %s", name), handle(name, config.Meters()))
	}

	for id, name := range site.Meters.PVMetersRef {
		if name != "" {
			d.DumpWithHeader(fmt.Sprintf("pv %d: %s", id+1, name), handle(name, config.Meters()))
		}
	}

	for id, name := range site.Meters.BatteryMetersRef {
		if name != "" {
			d.DumpWithHeader(fmt.Sprintf("battery %d: %s", id+1, name), handle(name, config.Meters()))
		}
	}

	for id, name := range site.Meters.AuxMetersRef {
		if name != "" {
			d.DumpWithHeader(fmt.Sprintf("aux %d: %s", id+1, name), handle(name, config.Meters()))
		}
	}

	for _, vv := range site.Vehicles().All() {
		v := vv.Instance()
		d.DumpWithHeader(fmt.Sprintf("vehicle: %s", v.Title()), v)
	}

	for id, lpI := range site.Loadpoints() {
		lp := lpI.(*core.Loadpoint)

		d.Header(fmt.Sprintf("loadpoint %d", id+1), "=")
		fmt.Println("")

		if name := lp.MeterRef; name != "" {
			d.DumpWithHeader(fmt.Sprintf("charge: %s", name), handle(name, config.Meters()))
		}

		if name := lp.ChargerRef; name != "" {
			d.DumpWithHeader(fmt.Sprintf("charger: %s", name), handle(name, config.Chargers()))
		}
	}
}
