package cmd

import (
	"net"
	"sync"

	"github.com/andig/evcc/hems/semp"
	"github.com/andig/evcc/util"
	"github.com/korylprince/ipnetgen"
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

type Task struct {
	ID, Type string
	Depends  string
	Config   map[string]interface{}
}

var taskList TaskList
var registry TaskHandlerRegistry = make(map[string]func(map[string]interface{}) (TaskHandler, error))

func init() {
	taskList = append(taskList, Task{
		ID:   "ping",
		Type: "ping",
	})

	taskList = append(taskList, Task{
		ID:      "tcp_502",
		Type:    "tcp",
		Depends: "ping",
		Config: map[string]interface{}{
			"port": 502,
		},
	})

	taskList = append(taskList, Task{
		ID:      "modbus_common",
		Type:    "modbus",
		Depends: "tcp_502",
		Config: map[string]interface{}{
			"min": 1,
			"max": 6,
		},
	})

	taskList = append(taskList, Task{
		ID:      "modbus_sma",
		Type:    "modbus",
		Depends: "tcp_502",
		Config: map[string]interface{}{
			"min": 126,
			"max": 126,
		},
	})

	taskList = append(taskList, Task{
		ID:      "modbus_kostal",
		Type:    "modbus",
		Depends: "tcp_502",
		Config: map[string]interface{}{
			"min": 71,
			"max": 71,
		},
	})

	taskList = append(taskList, Task{
		ID:      "tcp_1883",
		Type:    "tcp",
		Depends: "ping",
		Config: map[string]interface{}{
			"port": 1883,
		},
	})

	taskList = taskList.Sorted()
}

func worker(tasks <-chan net.IP) {
	for ip := range tasks {
		taskList.Test(ip)
	}
}

func runDetect(cmd *cobra.Command, args []string) {
	util.LogLevel("info", nil)

	ips := semp.LocalIPs()
	if len(ips) == 0 {
		log.FATAL.Fatal("could not find ip")
	}

	ip := ips[0]
	log.INFO.Println("my ip:", ip.IP)

	// subnet := ip.String()
	subnet := "192.168.0.201/24"
	log.INFO.Println("subnet:", subnet)

	gen, err := ipnetgen.New(subnet)
	if err != nil {
		log.FATAL.Fatal("could not create iterator")
	}

	tasks := make(chan net.IP)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			worker(tasks)
			wg.Done()
		}()
	}

	tasks <- net.ParseIP("127.0.0.1")

	for ip := gen.Next(); ip != nil; ip = gen.Next() {
		tasks <- ip
	}

	close(tasks)

	wg.Wait()
}
