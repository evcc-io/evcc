package tasks

import (
	"runtime"
	"time"

	"github.com/evcc-io/evcc/util"
	ping "github.com/prometheus-community/pro-bing"
)

const Ping TaskType = "ping"

func init() {
	registry.Add(Ping, PingHandlerFactory)
}

func PingHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := PingHandler{
		Count:   1,
		Timeout: timeout,
	}

	err := util.DecodeOther(conf, &handler)

	return &handler, err
}

type PingHandler struct {
	Count   int
	Timeout time.Duration
}

func (h *PingHandler) Test(log *util.Logger, in ResultDetails) []ResultDetails {
	pinger, err := ping.NewPinger(in.IP)
	if err != nil {
		panic(err)
	}

	if runtime.GOOS == "windows" {
		pinger.Size = 548 // https://github.com/go-ping/ping/issues/168
		pinger.SetPrivileged(true)
	}

	pinger.Count = h.Count
	pinger.Timeout = h.Timeout

	if err = pinger.Run(); err != nil {
		log.FATAL.Println("ping:", err)

		if runtime.GOOS != "windows" {
			log.FATAL.Println("")
			log.FATAL.Println("In order to run evcc in discovery mode, make sure to allow ping:")
			log.FATAL.Println("")
			log.FATAL.Println("	sudo sysctl -w net.ipv4.ping_group_range=\"0 2147483647\"")
		}

		log.FATAL.Fatalln("")
	}

	stat := pinger.Statistics()

	if stat.PacketsRecv == 0 {
		return nil
	}

	return []ResultDetails{in}
}
