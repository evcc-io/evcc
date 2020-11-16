package cmd

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/jq"
	"github.com/itchyny/gojq"
)

func init() {
	registry.Add("http", HttpHandlerFactory)
}

func HttpHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := HttpHandler{
		Schema: "http",
		Port:   80,
		Method: "GET",
		Codes:  []int{200},
		Header: map[string]string{
			"Content-type": "application/json",
		},
		Timeout: timeout,
	}

	err := util.DecodeOther(conf, &handler)

	if !(handler.Schema == "http" && handler.Port == 80 ||
		handler.Schema == "https" && handler.Port == 443) {
		handler.optionalPort = fmt.Sprintf(":%d", handler.Port)
	}

	if handler.Jq != "" {
		op, err := gojq.Parse(handler.Jq)
		if err != nil {
			return nil, fmt.Errorf("invalid jq query: %s (%s)", handler.Jq, err)
		}

		handler.query = op
	}

	return &handler, err
}

type HttpHandler struct {
	query                *gojq.Query
	Port                 int
	optionalPort         string
	Schema, Method, Path string
	Codes                []int
	Header               map[string]string
	Jq                   string
	Timeout              time.Duration
}

func (h *HttpHandler) Test(ip net.IP) bool {
	uri := fmt.Sprintf("%s://%s%s/%s", h.Schema, ip, h.optionalPort, strings.TrimLeft(h.Path, "/"))
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

	if len(h.Codes) > 0 {
		var status bool
		for _, code := range h.Codes {
			if resp.StatusCode == code {
				status = true
				break
			}
		}

		if !status {
			return false
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	if h.query == nil {
		return true
	}

	_, err = jq.Query(h.query, body)
	return err == nil
}
