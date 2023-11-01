package tasks

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/itchyny/gojq"
)

const Http TaskType = "http"

func init() {
	registry.Add(Http, HttpHandlerFactory)
}

type HttpResult struct {
	Jq interface{}
}

func HttpHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := HttpHandler{
		Schema: "http",
		Method: "GET",
		Codes:  []int{200},
		Header: map[string]string{
			"Content-type": "application/json",
		},
		Timeout: 3 * timeout,
	}

	err := util.DecodeOther(conf, &handler)

	switch handler.Schema {
	case "http":
		handler.Port = 80
	case "https":
		handler.Port = 443
	}

	if handler.Jq != "" {
		query, err := gojq.Parse(handler.Jq)
		if err != nil {
			return nil, fmt.Errorf("invalid jq query: %s (%s)", handler.Jq, err)
		}

		handler.query = query
	}

	return &handler, err
}

type HttpHandler struct {
	query                *gojq.Query
	Port                 int
	Schema, Method, Path string
	Codes                []int
	Header               map[string]string
	ResponseHeader       map[string]string
	Jq                   string
	Timeout              time.Duration
}

func (h *HttpHandler) Test(log *util.Logger, in ResultDetails) []ResultDetails {
	port := in.Port
	if port == 0 {
		port = h.Port
	}

	if port == 0 {
		panic("http: invalid port")
	}

	uri := fmt.Sprintf("%s://%s:%d/%s", h.Schema, in.IP, port, strings.TrimLeft(h.Path, "/"))
	req, err := http.NewRequest(strings.ToUpper(h.Method), uri, nil)
	if err != nil {
		return nil
	}

	client := http.Client{
		Timeout: h.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
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
			return nil
		}
	}

	for k, v := range h.ResponseHeader {
		if resp.Header.Get(k) != v {
			return nil
		}
	}

	var res HttpResult
	if h.query != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil
		}

		val, err := jq.Query(h.query, body)
		res.Jq = val

		if val == nil || err != nil {
			return nil
		}
	}

	if err == nil {
		out := in.Clone()
		out.Port = port
		return []ResultDetails{out}
	}

	return nil
}
