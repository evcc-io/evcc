package meter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/foogod/go-powerwall"
)

// PowerWall is the tesla powerwall meter
type PowerWall struct {
	usage  string
	client *powerwall.Client
}

func init() {
	registry.Add("tesla", NewPowerWallFromConfig)
	registry.Add("powerwall", NewPowerWallFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePowerWall -b *PowerWall -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,SoC,func() (float64, error)"

// NewPowerWallFromConfig creates a PowerWall Powerwall Meter from generic config
func NewPowerWallFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		URI, Usage, User, Password string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	if cc.Password == "" {
		return nil, errors.New("missing password")
	}

	// support default meter names
	switch strings.ToLower(cc.Usage) {
	case "grid":
		cc.Usage = "site"
	case "pv":
		cc.Usage = "solar"
	}

	return NewPowerWall(cc.URI, cc.Usage, cc.User, cc.Password)
}

// NewPowerWall creates a Tesla PowerWall Meter
func NewPowerWall(uri, usage, user, password string) (api.Meter, error) {
	log := util.NewLogger("tesla")

	httpClient := &http.Client{
		Transport: request.NewTripper(log, powerwall.DefaultTransport()),
		Timeout:   time.Second * 2, // Timeout after 2 seconds
	}

	client := powerwall.NewClient(uri, user, password, powerwall.WithHttpClient(httpClient))
	if _, err := client.GetStatus(); err != nil {
		return nil, err
	}

	m := &PowerWall{
		client: client,
		usage:  strings.ToLower(usage),
	}

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if m.usage == "load" || m.usage == "solar" {
		totalEnergy = m.totalEnergy
	}

	// decorate api.BatterySoC
	var batterySoC func() (float64, error)
	if usage == "battery" {
		batterySoC = m.batterySoC
	}

	return decoratePowerWall(m, totalEnergy, batterySoC), nil
}

var _ api.Meter = (*PowerWall)(nil)

// CurrentPower implements the api.Meter interface
func (m *PowerWall) CurrentPower() (float64, error) {
	res, err := m.client.GetMetersAggregates()
	if err != nil {
		return 0, err
	}

	if o, ok := res[m.usage]; ok {
		return float64(o.InstantPower), nil
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// totalEnergy implements the api.MeterEnergy interface
func (m *PowerWall) totalEnergy() (float64, error) {
	res, err := m.client.GetMetersAggregates()
	if err != nil {
		return 0, err
	}

	if o, ok := res[m.usage]; ok {
		switch m.usage {
		case "load":
			return float64(o.EnergyImported), nil
		case "solar":
			return float64(o.EnergyExported), nil
		}
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// batterySoC implements the api.Battery interface
func (m *PowerWall) batterySoC() (float64, error) {
	res, err := m.client.GetSOE()
	if err != nil {
		return 0, err
	}

	return float64(res.Percentage), err
}
