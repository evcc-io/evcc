package tasks

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logx"
	"github.com/go-kit/log/level"
	"github.com/go-ping/ping"
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

func (h *PingHandler) Test(log logx.Logger, in ResultDetails) []ResultDetails {
	pinger, err := ping.NewPinger(in.IP)
	if err != nil {
		panic(err)
	}

	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	}

	pinger.Count = h.Count
	pinger.Timeout = h.Timeout

	if err = pinger.Run(); err != nil {
		_ = level.Error(log).Log("msg", "ping", "error", err)

		if runtime.GOOS != "windows" {
			fmt.Println(`

	In order to run evcc in discovery mode, make sure to allow ping:

		sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"`)
		}

		os.Exit(1)
	}

	stat := pinger.Statistics()

	if stat.PacketsRecv == 0 {
		return nil
	}

	return []ResultDetails{in}
}
