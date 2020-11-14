package cmd

import (
	"github.com/andig/evcc/hems/semp"
	"github.com/spf13/cobra"

	"github.com/korylprince/ipnetgen"
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

func runDetect(cmd *cobra.Command, args []string) {
	ips := semp.LocalIPs()
	if len(ips) == 0 {
		log.FATAL.Fatal("could not find ip")
	}

	ip := ips[0].String()
	log.ERROR.Println("ip:", ip)

	gen, err := ipnetgen.New(ip + "/8")
	if err != nil {
		log.FATAL.Fatal("could not create iterator")
	}

	for ip := gen.Next(); ip != nil; ip = gen.Next() {
		//do something with ip
	}
}
