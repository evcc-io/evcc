package cmd

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strings"

	"github.com/andig/evcc/cmd/detect"
	"github.com/andig/evcc/hems/semp"
	"github.com/andig/evcc/util"
	"github.com/korylprince/ipnetgen"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// detectCmd represents the vehicle command
var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect compatible hardware",
	Run:   runDetect,
}

func init() {
	rootCmd.AddCommand(detectCmd)
}

func applicability(hit detect.Result) (string, string, string, string, string) {
	return "", "", "", "", ""
}

func IPsFromSubnet(arg string) (res []string) {
	// create ips
	gen, err := ipnetgen.New(arg)
	if err != nil {
		log.FATAL.Fatal("could not create iterator")
	}

	for ip := gen.Next(); ip != nil; ip = gen.Next() {
		res = append(res, ip.String())
	}

	return res
}

func ParseHostIPNet(arg string) (res []string) {
	if ip := net.ParseIP(arg); ip != nil {
		return []string{ip.String()}
	}

	_, ipnet, err := net.ParseCIDR(arg)

	// simple host
	if err != nil {
		return []string{arg}
	}

	// check subnet size
	if bits, _ := ipnet.Mask.Size(); bits < 24 {
		log.INFO.Println("skipping large subnet:", ipnet)
		return
	}

	return IPsFromSubnet(arg)
}

func runDetect(cmd *cobra.Command, args []string) {
	util.LogLevel("info", nil)

	var hosts []string

	// args
	for _, arg := range args {
		hosts = append(hosts, ParseHostIPNet(arg)...)
	}

	// autodetect
	if len(hosts) == 0 {
		ips := semp.LocalIPs()
		if len(ips) == 0 {
			log.FATAL.Fatal("could not find ip")
		}

		myIP := ips[0]
		log.INFO.Println("my ip:", myIP.IP)

		hosts = append(hosts, "127.0.0.1")
		hosts = append(hosts, IPsFromSubnet(myIP.String())...)
	}

	// magic happens here
	res := work(50, hosts)

	fmt.Println("")
	fmt.Println("SUMMARY")
	fmt.Println("-------")
	fmt.Println("")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"IP", "Hostname", "Task", "Details", "Charger", "Charge", "Grid", "PV", "Battery"})

	// use ip from SMA result
	for idx, hit := range res {
		if sma, ok := hit.Details.(detect.SmaResult); ok {
			hit.Host = sma.Addr
			res[idx] = hit
		}
	}

	// sort by host
	sort.Slice(res, func(i, j int) bool { return res[i].Host < res[j].Host })

	for _, hit := range res {
		switch hit.ID {
		case "ping", "tcp_80", "tcp_502", "sunspec":
			continue
		default:
			host := ""
			hosts, err := net.LookupAddr(hit.Host)
			if err == nil && len(hosts) > 0 {
				host = hosts[0]
				host = strings.TrimSuffix(host, ".")
			}

			details := ""
			if hit.Details != nil {
				details = fmt.Sprintf("%+v", hit.Details)
			}

			fmt.Printf("%-16s %-20s %-16s %s\n", hit.Host, host, hit.ID, details)

			charger, charge, grid, pv, battery := applicability(hit)

			table.Append([]string{
				hit.Host, host, hit.ID, details,
				charger, charge, grid, pv, battery,
			})
		}
	}

	fmt.Println("")
	table.Render()
}
