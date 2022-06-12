package idkproxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/urlvalues"
	"github.com/evcc-io/evcc/vehicle/vag"
)

const (
	BaseURL   = "https://idkproxy-service.apps.emea.vwapps.io"
	WellKnown = "https://idkproxy-service.apps.emea.vwapps.io/v1/emea/openid-configuration"
)

var Config = &oidc.ProviderConfig{
	AuthURL:  "https://identity.vwgroup.io/oidc/v1/authorize",
	TokenURL: "https://idkproxy-service.apps.emea.vwapps.io/v1/emea/token",
}

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

// https://github.com/arjenvrh/audi_connect_ha/issues/133

var secret = []byte{55, 24, 256 - 56, 256 - 96, 256 - 72, 256 - 110, 57, 256 - 87, 3, 256 - 86, 256 - 41, 256 - 103, 33, 256 - 30, 99, 103, 81, 125, 256 - 39, 256 - 39, 71, 18, 256 - 107, 256 - 112, 256 - 120, 256 - 12, 256 - 104, 89, 103, 113, 256 - 128, 256 - 91}

func qmauth(ts int64) string {
	hash := hmac.New(sha256.New, secret)
	hash.Write([]byte(strconv.FormatInt(ts, 10)))
	b := hash.Sum(nil)

	return hex.EncodeToString(b)
}

func qmauthNow() string {
	return "v1:55f755b0:" + qmauth(time.Now().Unix()/100)
}

// Exchange exchanges an VAG identity token for an IDK token
func (v *Service) Exchange(q url.Values) (*vag.Token, error) {
	if err := urlvalues.Require(q, "code", "code_verifier"); err != nil {
		return nil, err
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"response_type": {"token id_token"},
		"code":          {q.Get("code")},
		"code_verifier": {q.Get("code_verifier")},
	}

	urlvalues.Merge(data, v.data)

	var res vag.Token

	req, err := request.New(http.MethodPost, Config.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		"Accept":       request.JSONContent,
		"x-qmauth":     qmauthNow(),
	})
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// Refresh refreshes an IDK token
func (v *Service) Refresh(token *vag.Token) (*vag.Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"response_type": {"token id_token"},
		"refresh_token": {token.RefreshToken},
	}

	urlvalues.Merge(data, v.data)

	var res vag.Token

	req, err := request.New(http.MethodPost, Config.TokenURL, strings.NewReader(data.Encode()), map[string]string{
		"Content-Type": request.FormContent,
		"Accept":       request.JSONContent,
		"x-qmauth":     qmauthNow(),
	})
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// TokenSource creates token source. Token is refreshed automatically.
func (v *Service) TokenSource(token *vag.Token) vag.TokenSource {
	return vag.RefreshTokenSource(token, v.Refresh)
}
