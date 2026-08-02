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
	TotalEnergy() float64
	Currents() (float64, float64, float64)
	Voltages() (float64, float64, float64)
	Identify() string
}

type UpdateResponse map[string]any

type API interface {
	IsV2() bool
	Status() (Response, error)
	Update(payload string) error
}

type LocalAPI struct {
	*request.Helper
	uri     string
	v2      bool
	statusG util.Cacheable[Response]
}

var _ API = (*LocalAPI)(nil)

func NewLocal(log *util.Logger, uri string, cache time.Duration) *LocalAPI {
	uri = strings.TrimRight(uri, "/")
	uri = strings.TrimSuffix(uri, "/api")

	api := &LocalAPI{
		Helper: request.NewHelper(log),
		uri:    uri,
	}

	api.upgradeV2()
	api.statusG = util.ResettableCached(api.status, cache)

	return api
}

// upgradeV2 will switch to use the v2 api and revert if not available
func (c *LocalAPI) upgradeV2() {
	c.v2 = true // use v2 response struct

	res := new(StatusResponse2)
	err := c.response("api/status?filter=alw", &res)

	if err == nil {
		c.uri += "/api"
	} else {
		c.v2 = false
	}
}

// IsV2 returns v2 api usage
func (c *LocalAPI) IsV2() bool {
	return c.v2
}

// response returns a v1/v2 api response
func (c *LocalAPI) response(partial string, res any) error {
	url := fmt.Sprintf("%s/%s", c.uri, partial)
	return c.GetJSON(url, &res)
}

// status fetches a fresh v1/v2 api response
func (c *LocalAPI) status() (Response, error) {
	if c.v2 {
		var res StatusResponse2
		err := c.response("status?filter=alw,car,eto,nrg,wh,trx,cards", &res)
		return &res, err
	}

	var res StatusResponse
	err := c.response("status", &res)
	return &res, err
}

// Status reads a cached v1/v2 api response
func (c *LocalAPI) Status() (Response, error) {
	return c.statusG.Get()
}

// Update executes a v1/v2 api update and returns the response
func (c *LocalAPI) Update(payload string) error {
	c.statusG.Reset() // invalidate cache so the next Status refetches

	res := new(UpdateResponse)

	if c.v2 {
		return c.response(fmt.Sprintf("set?%s", payload), &res)
	}

	return c.response(fmt.Sprintf("mqtt?payload=%s", payload), &res)
}

type cloud struct {
	*request.Helper
	token   string
	statusG util.Cacheable[Response]
}

var _ API = (*cloud)(nil)

func NewCloud(log *util.Logger, token string, cache time.Duration) API {
	c := &cloud{
		Helper: request.NewHelper(log),
		token:  token,
	}
	c.statusG = util.ResettableCached(c.status, cache)

	return c
}

func (c *cloud) IsV2() bool {
	return false
}

func (c *cloud) response(function, payload string) (*StatusResponse, error) {
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

// status fetches a fresh cloud api response
func (c *cloud) status() (Response, error) {
	return c.response("api_status", "")
}

// Status reads a cached cloud api response
func (c *cloud) Status() (Response, error) {
	return c.statusG.Get()
}

func (c *cloud) Update(payload string) error {
	c.statusG.Reset() // invalidate cache so the next Status refetches

	_, err := c.response("api", payload)
	return err
}
