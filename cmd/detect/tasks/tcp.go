package tasks

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
)

const Tcp TaskType = "tcp"

func init() {
	registry.Add(Tcp, TcpHandlerFactory)
}

func TcpHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := TcpHandler{
		Timeout: timeout,
	}
	err := util.DecodeOther(conf, &handler)

	if err == nil && len(handler.Ports) == 0 {
		err = errors.New("missing port")
	}

	handler.dialer = net.Dialer{Timeout: handler.Timeout}
	return &handler, err
}

type TcpHandler struct {
	Ports   []int
	Timeout time.Duration
	dialer  net.Dialer
}

func (h *TcpHandler) Test(_ *util.Logger, in ResultDetails) (res []ResultDetails) {
	for _, port := range h.Ports {
		addr := net.JoinHostPort(in.IP, strconv.Itoa(port))
		conn, err := h.dialer.Dial("tcp", addr)
		if err == nil {
			defer conn.Close()
		}

		if err == nil {
			out := in.Clone()
			out.Port = port
			res = append(res, out)
		}
	}

	return res
}
