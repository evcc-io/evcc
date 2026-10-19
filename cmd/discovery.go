package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/service"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

// discoveryCmd represents the discovery command
var discoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Scan the local network for devices",
	Long: `Discovery scans the local network with the same unprivileged methods the configuration
UI uses for host suggestions (mDNS, SSDP, neighbor table, reverse DNS) and lists the hosts
found with their vendor and the configured devices they belong to.

The devices section is redacted (no addresses, vendor part of the MAC only, serial numbers
masked) and is meant to be shared on GitHub to improve the template discovery hints. The
hosts section is unredacted and stays local.`,
	Run: runDiscovery,
}

func init() {
	rootCmd.AddCommand(discoveryCmd)
	discoveryCmd.Flags().Bool(flagJSON, false, "Print the report as json")
}

const flagJSON = "json"

// newTable creates a table with a separator line between rows
func newTable(header ...string) *tablewriter.Table {
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Settings: tw.Settings{
				Separators: tw.Separators{BetweenRows: tw.On},
			},
		})),
	)
	table.Header(header)
	return table
}

func runDiscovery(cmd *cobra.Command, args []string) {
	jsonOutput := cmd.Flag(flagJSON).Changed

	// json output must stay parseable, keep the log quiet
	if jsonOutput {
		viper.Set("log", "error")
		util.LogLevel("error", nil)
	}

	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, &conf); err != nil {
		log.FATAL.Fatal(err)
	}

	configs := slices.Concat(conf.Meters, conf.Chargers, conf.Vehicles, conf.Curtailers)

	if !cmd.Flag(flagIgnoreDatabase).Changed {
		for _, class := range templates.ClassValues() {
			devs, err := config.ConfigurationsByClass(class)
			if err != nil {
				log.FATAL.Fatal(err)
			}
			for _, dev := range devs {
				configs = append(configs, dev.Named())
			}
		}
	}

	log.INFO.Println("scanning network...")
	report := service.Scan(cmd.Context(), configs)

	if jsonOutput {
		b, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(b))
		return
	}

	fmt.Println()
	hosts := newTable("IP", "MAC", "Vendor", "Hostname", "Services", "Used")
	for _, h := range report.Hosts {
		used := ""
		if h.Used {
			used = "✓"
		}
		names := h.Aliases
		if h.Hostname != "" {
			names = append([]string{h.Hostname}, h.Aliases...)
		}
		hosts.Append([]string{h.IP, h.Mac, h.Vendor, strings.Join(names, "\n"), strings.Join(h.Services, "\n"), used})
	}
	hosts.Render()

	if len(report.Devices) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Configured devices (redacted, safe to share):")
	devices := newTable("Template", "MAC", "Hostnames", "Services")
	for _, d := range report.Devices {
		devices.Append([]string{d.Template, d.Mac, strings.Join(d.Hostnames, "\n"), strings.Join(d.Services, "\n")})
	}
	devices.Render()
}
