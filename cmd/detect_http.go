package cmd

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/andig/evcc/util"
)

func init() {
	registry.Add("http", HttpHandlerFactory)
}

func HttpHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := HttpHandler{
		Schema: "http",
		Method: "GET",
		Codes:  []int{200},
		Header: map[string]string{
			"Content-type": "application/json",
		},
		Timeout: timeout,
	}

	err := util.DecodeOther(conf, &handler)

	return &handler, err
}

type HttpHandler struct {
	Schema, Method, Path string
	Codes                []int
	Header               map[string]string
	Timeout              time.Duration
}

func (h *HttpHandler) Test(ip net.IP) bool {

	uri := fmt.Sprintf("%s://%s/%s", h.Schema, ip, strings.TrimLeft(h.Path, "/"))
	req, err := http.NewRequest(strings.ToUpper(h.Method), uri, nil)
	if err != nil {
		return false
	}

	client := http.Client{
		Timeout: h.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	for _, code := range h.Codes {
		if resp.StatusCode == code {
			return true
		}
	}

	return false
}
