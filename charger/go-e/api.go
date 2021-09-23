package goe

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const CloudURI = "https://api.go-e.co"

// Response is the v1 and v2 api response interface
type Response interface {
	Status() int
	Enabled() bool
	CurrentPower() float64
	ChargedEnergy() float64
	Currents() (float64, float64, float64)
	Identify() string
}

type API interface {
	Status() (Response, error)
	Update(payload string) (Response, error)
}

type LocalAPI struct {
	*request.Helper
	uri string
	v2  bool
}

func NewLocal(log *util.Logger, uri string) *LocalAPI {
	uri = strings.TrimRight(uri, "/")
	uri = strings.TrimSuffix(uri, "/api")

	api := &LocalAPI{
		Helper: request.NewHelper(log),
		uri:    uri,
	}

	return api
}

func (c *LocalAPI) UpgradeV2() bool {
	c.v2 = true // use v2 response struct
	_, err := c.Response("api/status?filter=alw")

	if err != nil {
		c.v2 = false
	} else {
		c.uri = c.uri + "/api"
	}

	return err == nil
}

func (c *LocalAPI) Response(partial string) (Response, error) {
	var status Response
	if c.v2 {
		status = &StatusResponse2{}
	} else {
		status = &StatusResponse{}
	}

	url := fmt.Sprintf("%s/%s", c.uri, partial)
	err := c.GetJSON(url, &status)

	return status, err
}

func (c *LocalAPI) Status() (Response, error) {
	if c.v2 {
		return c.Response("status?filter=alw,car,eto,nrg,wh,trx,cards")
	}

	return c.Response("status")
}

func (c *LocalAPI) Update(payload string) (Response, error) {
	if c.v2 {
		return c.Response(fmt.Sprintf("set?%s", payload))
	}

	return c.Response(fmt.Sprintf("mqtt?payload=%s", payload))
}

type cloud struct {
	*request.Helper
	token   string
	cache   time.Duration
	updated time.Time
	status  Response
}

func NewCloud(log *util.Logger, token string, cache time.Duration) API {
	return &cloud{
		Helper: request.NewHelper(log),
		token:  token,
		cache:  cache,
	}
}

func (c *cloud) Response(function, payload string) (Response, error) {
	var status CloudResponse

	url := fmt.Sprintf("%s/%s?token=%s", CloudURI, function, c.token)
	if payload != "" {
		url += "&payload=" + payload
	}

	err := c.GetJSON(url, &status)
	if err == nil && status.Success != nil && !*status.Success {
		err = errors.New(status.Error)
	}

	return &status.Data, err
}

func (c *cloud) Status() (status Response, err error) {
	if time.Since(c.updated) >= c.cache {
		status, err = c.Response("api_status", "")
		if err == nil {
			c.updated = time.Now()
			c.status = status
		}
	}

	return c.status, err
}

func (c *cloud) Update(payload string) (Response, error) {
	c.updated = time.Time{}
	return c.Response("api", payload)
}
