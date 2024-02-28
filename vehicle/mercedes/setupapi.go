package mercedes

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type SetupAPI struct {
	log     *util.Logger
	account string
	region  string
	*request.Helper
}

func NewSetupAPI(log *util.Logger, account string, region string) *SetupAPI {
	client := request.NewHelper(log)

	client.Transport = &transport.Decorator{
		Base:      client.Transport,
		Decorator: transport.DecorateHeaders(mbheaders(true, region)),
	}

	return &SetupAPI{
		Helper:  client,
		log:     log,
		region:  region,
		account: account,
	}
}

func (vs *SetupAPI) RequestPin() (bool, *string, error) {
	client := request.NewHelper(vs.log)

	client.Transport = &transport.Decorator{
		Base:      client.Transport,
		Decorator: transport.DecorateHeaders(mbheaders(false, vs.region)),
	}

	// Preflight request required to get a pin
	uri := fmt.Sprintf("%s/v1/config", getBffUri(vs.region))
	if _, err := client.GetBody(uri); err != nil {
		return false, nil, err
	}

	nonce := uuid.New().String()
	data := PinRequest{
		EmailOrPhoneNumber: vs.account,
		CountryCode:        "EN",
		Nonce:              nonce,
	}

	uri = fmt.Sprintf("%s/v1/login", getBffUri(vs.region))
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data))
	if err != nil {
		return false, nil, err
	}

	var res PinResponse
	if err := client.DoJSON(req, &res); err != nil {
		return false, nil, err
	}

	// Only if the response field email is the same like the account an email is send by the servers.
	return res.UserName == vs.account, &nonce, nil
}

func (vs *SetupAPI) RequestAccessToken(nonce string, pin string) (*oauth2.Token, error) {
	data := url.Values{
		"client_id":  []string{ClientId},
		"grant_type": []string{"password"},
		"password":   []string{fmt.Sprintf("%s:%s", nonce, pin)},
		"scope":      []string{"openid email phone profile offline_access ciam-uid"},
		"username":   []string{vs.account},
	}

	uri := fmt.Sprintf("%s/as/token.oauth2", IdUri)
	req, _ := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), mbheaders(true, vs.region))

	var res oauth.Token
	if err := vs.DoJSON(req, &res); err != nil {
		return nil, err
	}

	return (*oauth2.Token)(&res), nil
}
