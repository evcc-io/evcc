package meter

import (
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/fritzbox"
	"github.com/evcc-io/evcc/util/request"
)

// AVM FritzBox AHA interface specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf

func init() {
	registry.Add("fritzdect", NewFritzDECTMeterFromConfig)
}

// NewFritzDECTMeterFromConfig creates a fritzdect meter from generic config
func NewFritzDECTMeterFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI          string
		AIN          string
		User         string
		Password     string
		SID          string
		StandbyPower float64
		Updated      time.Time
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		cc.URI = "https://fritz.box"
	}

	if cc.AIN == "" {
		return nil, errors.New("missing ain")
	}

	return NewFritzDECTMeter(cc.URI, cc.AIN, cc.User, cc.Password, cc.SID, cc.Updated)
}

// NewFritzDECTMeter creates FritzDECT meter
func NewFritzDECTMeter(uri, ain, user, password, sid string, updated time.Time) (*fritzbox.Connection, error) {
	log := util.NewLogger("fritzdect")

	m := &fritzbox.Connection{
		Helper:   request.NewHelper(log),
		URI:      strings.TrimRight(uri, "/"),
		AIN:      ain,
		User:     user,
		Password: password,
		SID:      sid,
	}

	m.Client.Transport = request.NewTripper(log, request.InsecureTransport())

	return m, nil
}
