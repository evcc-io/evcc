package cmd

import (
	"fmt"
	"net"
	"sync"
	"time"

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

const timeout = 100 * time.Millisecond

var (
	taskList                     = &TaskList{}
	registry TaskHandlerRegistry = make(map[string]func(map[string]interface{}) (TaskHandler, error))
)

func init() {
	taskList.Add(Task{
		ID:   "ping",
		Type: "ping",
	})

	taskList.Add(Task{
		ID:      "tcp_502",
		Type:    "tcp",
		Depends: "ping",
		Config: map[string]interface{}{
			"port": 502,
		},
	})

	taskList.Add(Task{
		ID:      "modbus",
		Type:    "modbus",
		Depends: "tcp_502",
		Config: map[string]interface{}{
			"ids":     []int{1, 2, 3, 4, 5, 6, 71, 126},
			"timeout": time.Second,
		},
	})

	taskList.Add(Task{
		ID:   "mqtt",
		Type: "mqtt",
	})

	taskList.Add(Task{
		ID:      "openwb",
		Type:    "mqtt",
		Depends: "mqtt",
		Config: map[string]interface{}{
			"topic": "openWB",
		},
	})

	taskList.Add(Task{
		ID:      "tcp_80",
		Type:    "tcp",
		Depends: "ping",
		Config: map[string]interface{}{
			"port": 80,
		},
	})

	taskList.Add(Task{
		ID:      "go-e",
		Type:    "http",
		Depends: "tcp_80",
		Config: map[string]interface{}{
			"path":    "/status",
			"timeout": 500 * time.Millisecond,
		},
	})

	taskList.Add(Task{
		ID:      "volkszähler",
		Type:    "http",
		Depends: "tcp_80",
		Config: map[string]interface{}{
			"path":    "/middleware.php/entity.json",
			"timeout": 500 * time.Millisecond,
		},
	})
}

func workers(num int, tasks <-chan net.IP) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			worker(tasks)
			wg.Done()
		}()
	}

	return &wg
}

func worker(tasks <-chan net.IP) {
	for ip := range tasks {
		taskList.Test(ip)
	}
}

func runDetect(cmd *cobra.Command, args []string) {
	util.LogLevel("info", nil)

	tasks := make(chan net.IP)
	wg := workers(50, tasks)

	ips := semp.LocalIPs()
	if len(ips) == 0 {
		log.FATAL.Fatal("could not find ip")
	}

	log.INFO.Println("my ip:", ips[0].IP)

	fmt.Println(args)
	if len(args) > 0 {
		ips = nil

		for _, arg := range args {
			_, ipnet, err := net.ParseCIDR(arg + "/32")
			if err != nil {
				log.FATAL.Fatal("could not parse", arg)
			}

			ips = append(ips, *ipnet)
		}
	} else {
		tasks <- net.ParseIP("127.0.0.1")
	}

	for _, ipnet := range ips {
		subnet := ipnet.String()

		if bits, _ := ipnet.Mask.Size(); bits < 24 {
			log.INFO.Println("skipping large subnet:", subnet)
			continue
		}

		log.INFO.Println("subnet:", subnet)

		gen, err := ipnetgen.New(subnet)
		if err != nil {
			log.FATAL.Fatal("could not create iterator")
		}

		for ip := gen.Next(); ip != nil; ip = gen.Next() {
			// log.INFO.Println("ip:", ip)
			tasks <- ip
		}
	}

	close(tasks)
	wg.Wait()
}
