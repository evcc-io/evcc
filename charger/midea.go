package charger

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/midea"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
)

// Midea charger implementation
type Midea struct {
	*embed
	*request.Helper
	hash       string
	session    string
	appliance  string
	applianceG provider.Cacheable[midea.Appliance]
}

func init() {
	registry.Add("midea", NewMideaFromConfig)
}

// NewMideaFromConfig creates a Midea charger from generic config
func NewMideaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User, Password       string
		HomeGroup, Appliance string
		Cache                time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMidea(cc.User, cc.Password, cc.HomeGroup, cc.Appliance, cc.Cache)
}

// NewMidea creates Midea charger
func NewMidea(user, password, homegroup, appliance string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("midea").Redact(user, password)

	c := &Midea{
		embed: &embed{
			Icon_:     "airpurifier", // TODO find better icon
			Features_: []api.Feature{api.IntegratedDevice},
		},
		Helper:    request.NewHelper(log),
		appliance: appliance,
	}

	var loginId midea.LoginId
	if err := mideaQuery(c.Helper, "/v1/user/login/id/get", map[string]string{
		"loginAccount": user,
	}, &loginId); err != nil {
		return nil, err
	}

	digest := sha256.Sum256([]byte(password))
	digest = sha256.Sum256([]byte(loginId.LoginId + hex.EncodeToString(digest[:]) + midea.AppKey))
	c.hash = hex.EncodeToString(digest[:])

	var login midea.Login
	if err := mideaQuery(c.Helper, "/v1/user/login", map[string]string{
		"loginAccount": user,
		"password":     c.hash,
	}, &login); err != nil {
		return nil, err
	}

	c.session = login.SessionId

	if homegroup == "" {
		var res midea.HomegroupList

		if err := mideaQuery(c.Helper, "/v1/homegroup/list/get", map[string]string{
			"sessionId": login.SessionId,
		}, &res); err != nil {
			return nil, err
		}

		if len(res.List) != 1 {
			return nil, fmt.Errorf("cannot find homegroup, got: %v", lo.Map(res.List, func(v midea.Homegroup, _ int) string {
				return v.Id
			}))
		}

		homegroup = res.List[0].Id
	}

	if appliance == "" {
		var res midea.ApplianceList

		if err := mideaQuery(c.Helper, "/v1/appliance/list/get", map[string]string{
			"sessionId":   c.session,
			"homegroupId": homegroup,
		}, &res); err != nil {
			return nil, err
		}

		if len(res.List) != 1 {
			return nil, fmt.Errorf("cannot find appliance, got: %v", lo.Map(res.List, func(v midea.Appliance, _ int) string {
				return v.Id
			}))
		}

		c.appliance = res.List[0].Id
	}

	c.applianceG = provider.ResettableCached(func() (midea.Appliance, error) {
		var res midea.ApplianceList

		err := mideaQuery(c.Helper, "/v1/appliance/list/get", map[string]string{
			"sessionId":   c.session,
			"homegroupId": homegroup,
		}, &res)

		idx := slices.IndexFunc(res.List, func(v midea.Appliance) bool {
			return v.Id == appliance
		})

		if idx < 0 {
			return midea.Appliance{}, errors.New("appliance not found")
		}

		return res.List[idx], err
	}, cache)

	return c, nil
}

func (c *Midea) Features() []api.Feature {
	return []api.Feature{api.IntegratedDevice}
}

func (c *Midea) Status() (api.ChargeStatus, error) {
	status := api.StatusA

	res, err := c.applianceG.Get()
	if err != nil {
		return status, err
	}

	if res.OnlineStatus == 1 {
		status = api.StatusB
	}

	if res.ActiveStatus == 1 {
		status = api.StatusC
	}

	return status, nil
}

func (c *Midea) Enable(enable bool) error {
	return nil
}

func (c *Midea) Enabled() (bool, error) {
	res, err := c.applianceG.Get()
	if err != nil {
		return false, err
	}

	return res.OnlineStatus == 1, nil
}

func (c *Midea) MaxCurrent(current int64) error {
	return nil
}

func mideaEncode(v url.Values) string {
	if len(v) == 0 {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(v)
		}
	}
	return buf.String()
}

func mideaSign(path string, v url.Values) {
	digest := sha256.Sum256([]byte(path + mideaEncode(v) + midea.AppKey))
	v["sign"] = []string{hex.EncodeToString(digest[:])}
}

func mideaQuery[T any](c *request.Helper, path string, query map[string]string, ret *T) error {
	res := struct {
		Msg       string
		ErrorCode int `json:",string"`
		Result    *T
	}{
		Result: ret,
	}

	data := url.Values{
		"appId":      {"1017"},
		"format":     {"2"},
		"clientType": {"1"},
		"language":   {"en_US"},
		"src":        {"17"},
		"stamp":      {time.Now().Format("20060102150405")},
	}

	for k, v := range query {
		data.Set(k, v)
	}

	mideaSign(path, data)

	req, _ := request.New(http.MethodPost, midea.BaseUrl+path, strings.NewReader(data.Encode()), request.URLEncoding)
	return c.DoJSON(req, &res)
}
