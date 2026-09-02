package mercedes

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"uuid"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	log       *util.Logger
	account   string
	region    string
	Sessionid string
}

// OAuth2Config is the OAuth2 configuration for authenticating with the MercedesAPI.
var OAuth2Config = &oauth2.Config{
	//	RedirectURL: fmt.Sprintf("%s/void/RedirectURL", IdUri),
	Endpoint: oauth2.Endpoint{
		// AuthURL:   fmt.Sprintf("%s/void/AuthURL", IdUri),
		TokenURL:  fmt.Sprintf("%s/as/token.oauth2", IdUri),
		AuthStyle: oauth2.AuthStyleInParams,
	},
	Scopes: []string{"not_needed", "handled", "elsewhere"},
}

// NewIdentity creates Mercedes identity
func NewIdentity(log *util.Logger, token *oauth2.Token, account string, region string) (*Identity, error) {
	// serialise instance handling
	mu.Lock()
	defer mu.Unlock()

	v := &Identity{
		Helper:  request.NewHelper(log),
		log:     log,
		account: account,
		region:  region,
	}

	v.Sessionid = uuid.New().String()
	v.Helper.Transport = &transport.Decorator{
		Base:      v.Helper.Transport, //.NewTripper(log, transport.Insecure()),
		Decorator: transport.DecorateHeaders(mbheaders(true, region)),
	}

	// reuse identity instance
	if instance := getInstance(account); instance != nil {
		v.log.DEBUG.Println("identity.NewIdentity - token found in instance store")
		return instance, nil
	}

	ts, err := oauth.PersistentTokenSource(log, v.settingsKey(), token, v.refreshToken)
	if err != nil {
		return nil, err
	}

	v.TokenSource = ts

	// add instance
	addInstance(account, v)

	return v, nil
}

func (v *Identity) settingsKey() string {
	return fmt.Sprintf("mercedes.%s-%s", v.account, v.region)
}

func (v *Identity) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}

	uri := fmt.Sprintf("%s/as/token.oauth2", IdUri)
	req, _ := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), mbheaders(true, v.region))

	var res oauth2.Token
	if err := v.DoJSON(req, &res); err != nil {
		return nil, err
	}

	return util.TokenWithExpiry(&res), nil
}
