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

func New(log *util.Logger, q url.Values) *Service {
	return &Service{
		Helper: request.NewHelper(log),
		data:   q,
	}
}

type vwExchangeTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	IdToken      string `json:"idToken"`
}

func (v *Service) Exchange(q url.Values) (*vag.Token, error) {
	if err := urlvalues.Require(q, "id_token", "code"); err != nil {
		return nil, err
	}

	type CodeExData struct {
		Code        string `json:"code,omitempty"`
		RedirectUri string `json:"redirectUri,omitempty"`
		Verifier    string `json:"verifier,omitempty"`
	}

	data := CodeExData{
		Code:        q.Get("code"),
		RedirectUri: "myskoda://redirect/login/",
		Verifier:    q.Get("code_verifier"),
	}

	//urlvalues.Merge(data, v.data, q)

	var exchRes vwExchangeTokenResponse

	resp, err := v.Post(CodeExchangeURL, request.JSONContent, request.MarshalJSON(data))

	if err == nil {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&exchRes)
	}

	// req, err := request.New(http.MethodPost, CodeExchangeURL, strings.NewReader(data.Encode()), request.URLEncoding)
	// if err == nil {
	// 	err = v.DoJSON(req, &res)
	// }

	var res vag.Token

	res.AccessToken = exchRes.AccessToken
	res.RefreshToken = exchRes.RefreshToken
	res.IDToken = exchRes.IdToken

	return &res, err
}

func (v *Service) Refresh(token *vag.Token) (*vag.Token, error) {

	type RefreshData struct {
		Token string `json:"token"`
	}

	data := RefreshData{
		Token: token.RefreshToken,
	}

	var refreshResp vwExchangeTokenResponse

	resp, err := v.Post(RefreshTokenURL, request.JSONContent, request.MarshalJSON(data))

	if err == nil {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&refreshResp)
	}

	// req, err := request.New(http.MethodPost, CodeExchangeURL, strings.NewReader(data.Encode()), request.URLEncoding)
	// if err == nil {
	// 	err = v.DoJSON(req, &res)
	// }

	var res vag.Token

	res.AccessToken = refreshResp.AccessToken
	res.RefreshToken = refreshResp.RefreshToken
	res.IDToken = refreshResp.IdToken
	res.Expiry = time.Now().Add(time.Minute * 60)

	return &res, err
}

// TokenSource creates token source. Token is refreshed automatically.
func (v *Service) TokenSource(token *vag.Token) vag.TokenSource {
	return vag.RefreshTokenSource(token, v.Refresh)
}
