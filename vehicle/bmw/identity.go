package bmw

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/oauth"
	"github.com/andig/evcc/util/request"
	"golang.org/x/oauth2"
)

const AuthURI = "https://customer.bmwgroup.com/gcdm/oauth/authenticate"

type Identity struct {
	*request.Helper
	log *util.Logger
	oauth2.TokenSource
	user, password string
}

// NewIdentity creates BMW identity
func NewIdentity(log *util.Logger) *Identity {
	v := &Identity{
		log:    log,
		Helper: request.NewHelper(log),
	}

	return v
}

func (v *Identity) Login(user, password string) error {
	v.user = user
	v.password = password

	token, err := v.RefreshToken(nil)

	if err == nil {
		v.TokenSource = oauth.RefreshTokenSource(token, v)
	}

	return err
}

func (v *Identity) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	data := url.Values{
		"username":      []string{v.user},
		"password":      []string{v.password},
		"client_id":     []string{"dbf0a542-ebd1-4ff0-a9a7-55172fbfce35"},
		"redirect_uri":  []string{"https://www.bmw-connecteddrive.com/app/default/static/external-dispatch.html"},
		"response_type": []string{"token"},
		"scope":         []string{"authenticate_user fupo"},
		"state":         []string{"eyJtYXJrZXQiOiJkZSIsImxhbmd1YWdlIjoiZGUiLCJkZXN0aW5hdGlvbiI6ImxhbmRpbmdQYWdlIn0"},
		"locale":        []string{"DE-de"},
	}

	req, err := request.New(http.MethodPost, AuthURI, strings.NewReader(data.Encode()), request.URLEncoding)
	if err != nil {
		return nil, err
	}

	// don't follow redirects
	v.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	defer func() { v.Client.CheckRedirect = nil }()

	resp, err := v.Do(req)
	if err != nil {
		return nil, err
	}

	query, err := url.ParseQuery(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	at := query.Get("access_token")
	expires, err := strconv.Atoi(query.Get("expires_in"))
	if err != nil || at == "" || expires == 0 {
		return nil, errors.New("could not obtain token")
	}

	token := &oauth2.Token{
		AccessToken: at,
		Expiry:      time.Now().Add(time.Duration(expires) * time.Second),
	}

	return token, nil
}
