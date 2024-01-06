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
	"github.com/evcc-io/evcc/vehicle/vag/cariad"
)

const WellKnown = cariad.BaseURL + "/login/v1/idk/openid-configuration"

var Config = &oidc.ProviderConfig{
	AuthURL:  "https://identity.vwgroup.io/oidc/v1/authorize",
	TokenURL: cariad.BaseURL + "/login/v1/idk/token",
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

const (
	qmSecret   = "e47866378ef0658ce75d71007a809f34616b9635e2ec228245784c1f63e88d06"
	qmClientId = "c95f4fd2"
)

func qmauth(ts int64) string {
	secret, _ := hex.DecodeString(qmSecret)
	hash := hmac.New(sha256.New, secret)
	hash.Write([]byte(strconv.FormatInt(ts, 10)))
	return hex.EncodeToString(hash.Sum(nil))
}

func qmauthNow() string {
	return "v1:" + qmClientId + ":" + qmauth(time.Now().Unix()/100)
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
