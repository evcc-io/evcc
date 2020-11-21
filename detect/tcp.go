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

	if err == nil && handler.Port == 0 {
		err = errors.New("missing port")
	}

	handler.dialer = net.Dialer{Timeout: handler.Timeout}
	return &handler, err
}

type TcpHandler struct {
	Port    int
	Timeout time.Duration
	dialer  net.Dialer
}

func (h *TcpHandler) Test(log *util.Logger, ip string) []interface{} {
	addr := fmt.Sprintf("%s:%d", ip, h.Port)
	conn, err := h.dialer.Dial("tcp", addr)
	if err == nil {
		defer conn.Close()
	}

	if err == nil {
		return []interface{}{nil}
	}

	return nil
}
