package vehicle

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/seat"
	"github.com/evcc-io/evcc/vehicle/seat/cupra"
	"github.com/evcc-io/evcc/vehicle/vag/service"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"github.com/evcc-io/evcc/vehicle/vw"
	"golang.org/x/oauth2"
)

// https://github.com/trocotronic/weconnect
// https://github.com/TA2k/ioBroker.vw-connect

// Seat is an api.Vehicle implementation for Seat cars
type Seat struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("seat", NewSeatFromConfig)
}

// NewSeatFromConfig creates a new vehicle
func NewSeatFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &Seat{
		embed: &cc.embed,
	}

	log := util.NewLogger("seat").Redact(cc.User, cc.Password, cc.VIN)

	trs, err := service.TokenRefreshServiceTokenSource(log, cupra.TRSParams, cupra.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	// get OIDC user information
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))
	ui, err := vwidentity.Config.NewProvider(ctx).UserInfo(ctx, trs)
	if err != nil {
		return nil, fmt.Errorf("failed getting user information: %w", err)
	}

	mbbId, err := mbbUserId(log, trs, ui.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed getting mbbUserId: %w", err)
	}

	ts := service.MbbTokenSource(log, trs, seat.AuthClientID)
	api := seat.NewAPI(log, ts)

	cc.VIN, err = ensureVehicle(
		cc.VIN, func() ([]string, error) {
			return api.Vehicles(mbbId)
		},
	)

	if err == nil {
		api := vw.NewAPI(log, ts, seat.Brand, seat.Country)
		api.Client.Timeout = cc.Timeout

		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
		}
	}

	return v, err
}

func mbbUserId(log *util.Logger, ts oauth2.TokenSource, uid string) (string, error) {
	client := request.NewHelper(log)
	client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   client.Transport,
	}

	data := url.Values{
		"scopeId": []string{"commonMandatoryFields"},
	}

	var mandatoryConsentInfo struct {
		MbbUserId string `json:"mbbUserId"`
	}

	uri := fmt.Sprintf("https://profileintegrityservice.apps.emea.vwapps.io/iaa/pic/v1/users/%s/check-profile", uid)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)

	if err == nil {
		err = client.DoJSON(req, &mandatoryConsentInfo)
	}

	return mandatoryConsentInfo.MbbUserId, err
}
