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

const WellKnown = cariad.BaseURL + "/auth/v1/idk/oidc/openid-configuration"

var Config = &oidc.ProviderConfig{
	AuthURL:  "https://identity.vwgroup.io/oidc/v1/authorize",
	TokenURL: cariad.BaseURL + "/auth/v1/idk/oidc/token",
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
	qmSecret   = "1ab69925ac179aaa4e83abe671a9476d176418b85bd706f1436ca15be647989c"
	qmClientId = "01da27b0"
	userAgent  = "Android/4.31.0 (Build 800341641.root project 'myaudi_android'.ext.buildTime) Android/13"
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

// tokenHeaders returns the assertion header set the IDK token endpoint
// validates for both authorization_code and refresh_token grants.
func tokenHeaders() map[string]string {
	return map[string]string{
		"Content-Type":           request.FormContent,
		"Accept":                 request.JSONContent,
		"Accept-Charset":         "utf-8",
		"User-Agent":             userAgent,
		"x-qmauth":               qmauthNow(),
		"x-platform":             "android",
		"x-android-package-name": "de.myaudi.mobile.assistant",
		"x-assertion":            "0",
	}
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

	req, err := request.New(http.MethodPost, Config.TokenURL, strings.NewReader(data.Encode()), tokenHeaders())
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

	req, err := request.New(http.MethodPost, Config.TokenURL, strings.NewReader(data.Encode()), tokenHeaders())
	if err == nil {
		err = v.DoJSON(req, &res)
	}

	return &res, err
}

// TokenSource creates token source. Token is refreshed automatically.
func (v *Service) TokenSource(token *vag.Token) vag.TokenSource {
	return vag.RefreshTokenSource(token, v.Refresh)
}
