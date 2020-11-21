package detect

import (
	"fmt"
	"io/ioutil"
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

type HttpResult struct {
	Jq interface{}
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
		Timeout: 3 * timeout,
	}

	err := util.DecodeOther(conf, &handler)

	if !(handler.Schema == "http" && handler.Port == 80 ||
		handler.Schema == "https" && handler.Port == 443) {
		handler.optionalPort = fmt.Sprintf(":%d", handler.Port)
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
	optionalPort         string
	Schema, Method, Path string
	Codes                []int
	Header               map[string]string
	Jq                   string
	Timeout              time.Duration
}

func (h *HttpHandler) Test(log *util.Logger, ip string) []interface{} {
	uri := fmt.Sprintf("%s://%s%s/%s", h.Schema, ip, h.optionalPort, strings.TrimLeft(h.Path, "/"))
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var res HttpResult
	if h.query != nil {
		val, err := jq.Query(h.query, body)
		res.Jq = val

		if val == nil || err != nil {
			return nil
		}
	}

	if err == nil {
		return []interface{}{res}
	}

	return nil
}
