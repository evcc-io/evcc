package charger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/andig/evcc/api"
)

type apiFunction string

// NewFromConfig creates charger from configuration
func NewFromConfig(log *api.Logger, typ string, other map[string]interface{}) api.Charger {
	var c api.Charger

	switch strings.ToLower(typ) {
	case "wallbe":
		c = NewWallbeFromConfig(log, other)
	case "phoenix":
		c = NewPhoenixFromConfig(log, other)
	case "nrgkick", "nrg", "kick":
		c = NewNRGKickFromConfig(log, other)
	case "go-e", "goe":
		c = NewGoEFromConfig(log, other)
	case "simpleevse", "evse":
		c = NewSimpleEVSEFromConfig(log, other)
	case "default", "configurable":
		c = NewConfigurableFromConfig(log, other)
	default:
		log.FATAL.Fatalf("invalid charger type '%s'", typ)
	}

	return c
}

func getJSON(url string, result interface{}) (*http.Response, []byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return resp, []byte{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, body, err
	}

	if resp.StatusCode == http.StatusOK {
		err = json.Unmarshal(body, &result)
		return resp, body, err
	}

	return resp, body, fmt.Errorf("unexpected status %d", resp.StatusCode)
}

func putJSON(url string, request interface{}) (*http.Response, []byte, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, []byte{}, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, []byte{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, []byte{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return resp, body, err
}
