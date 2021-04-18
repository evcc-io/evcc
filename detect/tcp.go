package detect

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/andig/evcc/util"
)

func init() {
	registry.Add("tcp", TcpHandlerFactory)
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

func (h *TcpHandler) Test(log *util.Logger, in Details) (res []Details) {
	for _, port := range h.Ports {
		addr := fmt.Sprintf("%s:%d", in.IP, port)
		conn, err := h.dialer.Dial("tcp", addr)
		if err == nil {
			defer conn.Close()
		}

		if err == nil {
			out := in
			out.Port = port
			res = append(res, out)
		}
	}

	fmt.Println("tcp", h.Ports, res)
	return res
}
