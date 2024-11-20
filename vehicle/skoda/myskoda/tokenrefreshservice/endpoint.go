package tokenrefreshservice

import (
	"encoding/json"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
)

const (
	BaseURL         = "https://mysmob.api.connect.skoda-auto.cz"
	CodeExchangeURL = BaseURL + "/api/v1/authentication/exchange-authorization-code?tokenType=CONNECT"
	RefreshTokenURL = BaseURL + "/api/v1/authentication/refresh-token?tokenType=CONNECT"
)

var _ vag.TokenExchanger = (*Service)(nil)

type Service struct {
	*request.Helper
	data url.Values
}

type skodaTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	IdToken      string `json:"idToken"`
}

func (s *skodaTokenResponse) toVagToken(v *vag.Token) {
	v.AccessToken = s.AccessToken
	v.RefreshToken = s.RefreshToken
	v.IDToken = s.IdToken
	v.Expiry = time.Now().Add(time.Minute * 60)
}

func New(log *util.Logger, q url.Values) *Service {
	return &Service{
		Helper: request.NewHelper(log),
		data:   q,
	}
}

func (v *Service) Exchange(q url.Values) (*vag.Token, error) {
	if err := urlvalues.Require(q, "id_token", "code"); err != nil {
		return nil, err
	}

	var skTok skodaTokenResponse

	data := map[string]string{
		"code":        q.Get("code"),
		"redirectUri": "myskoda://redirect/login/",
		"verifier":    q.Get("code_verifier"),
	}

	resp, err := v.Post(CodeExchangeURL, request.JSONContent, request.MarshalJSON(data))

	if err == nil {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&skTok)
	}

	var res vag.Token
	skTok.toVagToken(&res)
	return &res, err
}

func (v *Service) Refresh(token *vag.Token) (*vag.Token, error) {
	var skTok skodaTokenResponse

	data := map[string]string{"token": token.RefreshToken}
	resp, err := v.Post(RefreshTokenURL, request.JSONContent, request.MarshalJSON(data))

	if err == nil {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&skTok)
	}

	var res vag.Token
	skTok.toVagToken(&res)
	return &res, err
}

// TokenSource creates token source. Token is refreshed automatically.
func (v *Service) TokenSource(token *vag.Token) vag.TokenSource {
	return vag.RefreshTokenSource(token, v.Refresh)
}
