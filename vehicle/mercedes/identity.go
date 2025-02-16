package mercedes

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type Identity struct {
	*request.Helper
	oauth2.TokenSource
	mu        sync.Mutex
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

	v.log.DEBUG.Println("identity.NewIdentity - refreshToken started")
	if tok, err := v.RefreshToken(token); err == nil {
		token = tok
	}

	if !token.Valid() {
		return nil, errors.New("token expired. Update your token in your configuration.")
	}

	v.TokenSource = oauth.RefreshTokenSource(token, v)

	return v, nil
}

func (v *Identity) RefreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

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

	tok := util.TokenWithExpiry(&res)
	tok.RefreshToken = token.RefreshToken
	v.TokenSource = oauth.RefreshTokenSource(tok, v)

	return tok, nil
}
