package tesla

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const authURL = "https://auth.tesla.com/oauth2/v3/authorize"

var (
	tokenURL = "https://fleet-auth.prd.vn.cloud.tesla.com/oauth2/v3/token"

	// the account region is unknown before login, register everywhere
	fleetAudiences = []string{teslaclient.FleetAudienceEU, teslaclient.FleetAudienceNA}
)

// tesla binds a public key to a single app and domain
var errKeyTaken = errors.New("public key already registered for another app or domain")

// registerPartner registers the domain hosting the public key with the Fleet API.
// Tesla fetches the key from the domain during this call. The call is idempotent.
func registerPartner(ctx context.Context, log *util.Logger, clientID, clientSecret, domain string) error {
	var errs []error

	for _, audience := range fleetAudiences {
		if err := registerPartnerRegion(ctx, log, clientID, clientSecret, domain, audience); err != nil {
			log.DEBUG.Printf("register %s with %s: %v", domain, audience, err)
			errs = append(errs, err)
		}
	}

	// the account's vehicles live in one region
	if len(errs) < len(fleetAudiences) {
		return nil
	}

	for _, err := range errs {
		switch {
		case strings.Contains(err.Error(), "Public key hash has already been taken"):
			return errKeyTaken
		case strings.Contains(err.Error(), "Domain has already been taken"):
			return fmt.Errorf("%s is registered with another Tesla app, use that app's client id and secret", domain)
		}
	}

	return errs[0]
}

func registerPartnerRegion(ctx context.Context, log *util.Logger, clientID, clientSecret, domain, audience string) error {
	cc := clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TokenURL:       tokenURL,
		Scopes:         []string{"openid", "vehicle_device_data", "vehicle_cmds", "vehicle_charging_cmds"},
		EndpointParams: url.Values{"audience": {strings.TrimSuffix(audience, "/")}},
		AuthStyle:      oauth2.AuthStyleInParams,
	}

	helper := request.NewHelper(log)
	helper.Transport = &oauth2.Transport{
		Source: cc.TokenSource(context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))),
		Base:   helper.Transport,
	}

	req, err := request.New(http.MethodPost, audience+"api/1/partner_accounts", request.MarshalJSON(map[string]string{
		"domain": domain,
	}), request.JSONEncoding)
	if err != nil {
		return err
	}

	if _, err := helper.DoBody(req); err != nil {
		var se *request.StatusError
		if errors.As(err, &se) {
			return fmt.Errorf("register partner: %w: %s", err, strings.TrimSpace(string(se.Body())))
		}
		return fmt.Errorf("register partner: %w", err)
	}

	log.DEBUG.Printf("registered %s with %s", domain, audience)

	return nil
}
