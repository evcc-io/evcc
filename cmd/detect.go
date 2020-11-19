package cmd

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/andig/evcc/detect"
	"github.com/andig/evcc/util"
	"github.com/korylprince/ipnetgen"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// detectCmd represents the vehicle command
var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Auto-detect compatible hardware",
	Run:   runDetect,
}

func init() {
	rootCmd.AddCommand(detectCmd)
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

func display(res []detect.Result) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"IP", "Hostname", "Task", "Details"})
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)

	for _, hit := range res {
		switch hit.ID {
		case detect.TaskPing, detect.TaskTCP80, detect.TaskTCP502:
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

			// fmt.Printf("%-16s %-20s %-16s %s\n", hit.Host, host, hit.ID, details)
			table.Append([]string{hit.Host, host, hit.ID, details})
		}
	}

	fmt.Println("")
	table.Render()

	fmt.Println(`
Please open https://github.com/andig/evcc/issues/new in your browser and copy the
results above into a new issue. Please tell us:

	1. Is the scan result correct?
	2. If not correct: please describe your hardware setup.`)
	fmt.Println("")
}

func runDetect(cmd *cobra.Command, args []string) {
	util.LogLevel("info", nil)

	// args
	var hosts []string
	for _, arg := range args {
		hosts = append(hosts, ParseHostIPNet(arg)...)
	}

	// autodetect
	if len(hosts) == 0 {
		ips := util.LocalIPs()
		if len(ips) == 0 {
			log.FATAL.Fatal("could not find ip")
		}

		myIP := ips[0]
		log.INFO.Println("my ip:", myIP.IP)

		hosts = append(hosts, "127.0.0.1")
		hosts = append(hosts, IPsFromSubnet(myIP.String())...)
	}

	// magic happens here
	res := detect.Work(log, 50, hosts)
	// res := []detect.Result{
	// 	{
	// 		Task: detect.Task{
	// 			ID:   "sma",
	// 			Type: "sma",
	// 		},
	// 		Host: "server",
	// 		Details: detect.SmaResult{
	// 			Addr:   "sem",
	// 			Serial: "0815",
	// 			Http:   true,
	// 		},
	// 	}, {
	// 		Task: detect.Task{
	// 			ID:   "sma",
	// 			Type: "sma",
	// 		},
	// 		Host: "server",
	// 		Details: detect.SmaResult{
	// 			Addr:   "sem-2",
	// 			Serial: "0815-2",
	// 			Http:   true,
	// 		},
	// 	}, {
	// 		Task: detect.Task{
	// 			ID:   "sma",
	// 			Type: "sma",
	// 		},
	// 		Host: "server",
	// 		Details: detect.SmaResult{
	// 			Addr:   "shm",
	// 			Serial: "4711",
	// 			Http:   false,
	// 		},
	// 	}, {
	// 		Task: detect.Task{
	// 			ID:   "modbus_inverter",
	// 			Type: "modbus",
	// 		},
	// 		Host: "wr",
	// 		Details: detect.ModbusResult{
	// 			SlaveID: 126,
	// 		},
	// 	},
	// }

	display(res)

	// sum := detect.Consolidate(res)
	// fmt.Printf("%+v\n", sum)
}
