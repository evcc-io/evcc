package chargepoint

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// wsUserAgent is the User-Agent for webservices.chargepoint.com calls,
// matching the iOS app's WKWebView requests.
const wsUserAgent = "ChargePoint/664 (iPhone; iOS 26.3; Scale/3.00)"

// API is an HTTP client for the ChargePoint API.
type API struct {
	log         *util.Logger
	identity    *Identity
	wsURL       string
	accountsURL string
	internalURL string
	chargersURL string
	region      string
}

// NewAPI creates a ChargePoint API client.
func NewAPI(log *util.Logger, identity *Identity) *API {
	return &API{
		log:         log,
		identity:    identity,
		wsURL:       identity.cfg.EndPoints.WebServices.Value,
		accountsURL: identity.cfg.EndPoints.Accounts.Value,
		internalURL: identity.cfg.EndPoints.InternalAPI.Value,
		chargersURL: identity.cfg.EndPoints.Chargers.Value,
		region:      identity.Region,
	}
}

// cpHeaders returns the standard CP headers required by all API endpoints.
// Cookies are set explicitly because the cookie jar is empty after a settings
// restore and the app always sends them as static header values.
func (a *API) cpHeaders() map[string]string {
	// Accept-Encoding is intentionally omitted here; the BrotliCompression
	// transport (set in NewIdentity) sets it to "br" on every request.
	return map[string]string{
		"User-Agent":       userAgent,
		"CP-Region":        a.region,
		"CP-Session-Token": a.identity.SessionID,
		"CP-Session-Type":  "CP_SESSION_TOKEN",
		"Cache-Control":    "no-store",
		"Accept-Language":  "en;q=1",
		"Cookie":           "coulomb_sess=" + a.identity.SessionID + "; auth-session=" + a.identity.SSOSessionID,
	}
}

// cpWSHeaders returns CP headers for webservices.chargepoint.com calls,
// which require a different User-Agent from the native app endpoints.
func (a *API) cpWSHeaders() map[string]string {
	h := a.cpHeaders()
	h["User-Agent"] = wsUserAgent
	return h
}

// cpInternalHeaders returns CP headers for internal-api calls, which
// additionally require an Authorization bearer token.
func (a *API) cpInternalHeaders() map[string]string {
	h := a.cpHeaders()
	h["Authorization"] = "Bearer " + a.identity.SSOSessionID
	return h
}

// doJSON executes the request produced by makeReq. If the server returns 401,
// it re-authenticates and retries once with a freshly-built request.
func (a *API) doJSON(makeReq func() (*http.Request, error), res any) error {
	req, err := makeReq()
	if err != nil {
		return err
	}
	err = a.identity.DoJSON(req, res)
	if err == nil {
		return nil
	}
	var se *request.StatusError
	if !errors.As(err, &se) || !se.HasStatus(http.StatusUnauthorized) {
		return err
	}
	// Session expired — re-authenticate and retry once.
	a.log.DEBUG.Println("chargepoint session expired, re-authenticating")
	if loginErr := a.identity.Login(); loginErr != nil {
		return fmt.Errorf("re-authentication failed: %w (original: %v)", loginErr, err)
	}
	req, err = makeReq()
	if err != nil {
		return err
	}
	return a.identity.DoJSON(req, res)
}

// Account fetches the account and returns the user ID.
func (a *API) Account() (int32, error) {
	var res struct {
		User struct {
			UserID int32 `json:"userId"`
		} `json:"user"`
	}
	err := a.doJSON(func() (*http.Request, error) {
		return request.New(http.MethodGet, a.accountsURL+"v1/driver/profile/user", nil,
			request.JSONEncoding, a.cpHeaders())
	}, &res)
	if err != nil {
		return 0, err
	}
	return res.User.UserID, nil
}

// HomeChargerIDs returns the device IDs of all registered home chargers.
func (a *API) HomeChargerIDs() ([]int, error) {
	data := struct {
		UserID    int32 `json:"user_id"`
		GetPandas struct {
			MFHS struct{} `json:"mfhs"`
		} `json:"get_pandas"`
	}{UserID: a.identity.UserID}

	var res struct {
		GetPandas struct {
			DeviceIDs []int `json:"device_ids"`
		} `json:"get_pandas"`
	}
	err := a.doJSON(func() (*http.Request, error) {
		return request.New(http.MethodPost, a.wsURL+"mobileapi/v5",
			request.MarshalJSON(data), request.JSONEncoding, a.cpWSHeaders())
	}, &res)
	if err != nil {
		return nil, err
	}

	return res.GetPandas.DeviceIDs, nil
}

// HomeChargerStatus returns the current status of a home charger via the
// internal REST API, which returns richer data than the legacy mobileapi.
func (a *API) HomeChargerStatus(deviceID int) (HomeChargerStatus, error) {
	uri := fmt.Sprintf("%sapi/v1/configuration/users/%d/chargers/%d/status?", a.chargersURL, a.identity.UserID, deviceID)

	var res HomeChargerStatus
	err := a.doJSON(func() (*http.Request, error) {
		return request.New(http.MethodGet, uri, nil,
			request.JSONEncoding, a.cpInternalHeaders())
	}, &res)
	return res, err
}

// StartSession starts a charging session on the given device.
func (a *API) StartSession(deviceID int) error {
	data := struct {
		DeviceData DeviceData `json:"deviceData"`
		DeviceID   int        `json:"deviceId"`
	}{
		DeviceData: a.identity.deviceData,
		DeviceID:   deviceID,
	}

	var res struct {
		AckID int `json:"ackId"`
	}
	if err := a.doJSON(func() (*http.Request, error) {
		return request.New(http.MethodPost, a.accountsURL+"v1/driver/station/startsession",
			request.MarshalJSON(data), request.JSONEncoding, a.cpHeaders())
	}, &res); err != nil {
		// 422 means the charger received the command but responds with an ack ID
		// in the body — decodeJSON still populates res on error, so fall through
		// to poll. Any other error is fatal.
		var se *request.StatusError
		if !errors.As(err, &se) || !se.HasStatus(http.StatusUnprocessableEntity) {
			return err
		}
	}

	return a.pollAck(res.AckID, "start_session")
}

// StopSession stops the active charging session on the given device.
func (a *API) StopSession(deviceID int) error {
	data := struct {
		DeviceData DeviceData `json:"deviceData"`
		DeviceID   int        `json:"deviceId"`
	}{
		DeviceData: a.identity.deviceData,
		DeviceID:   deviceID,
	}

	var res struct {
		AckID int `json:"ackId"`
	}
	if err := a.doJSON(func() (*http.Request, error) {
		return request.New(http.MethodPost, a.accountsURL+"v1/driver/station/stopsession",
			request.MarshalJSON(data), request.JSONEncoding, a.cpHeaders())
	}, &res); err != nil {
		// 422 means the charger received the command but responds with an ack ID
		// in the body — decodeJSON still populates res on error, so fall through
		// to poll. Any other error is fatal.
		var se *request.StatusError
		if !errors.As(err, &se) || !se.HasStatus(http.StatusUnprocessableEntity) {
			return err
		}
	}

	return a.pollAck(res.AckID, "stop_session")
}

func (a *API) pollAck(ackID int, action string) error {
	ackData := struct {
		DeviceData DeviceData `json:"deviceData"`
		AckID      int        `json:"ackId"`
		Action     string     `json:"action"`
	}{
		DeviceData: a.identity.deviceData,
		AckID:      ackID,
		Action:     action,
	}

	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(time.Second)
		}

		req, err := request.New(http.MethodPost, a.accountsURL+"v1/driver/station/session/ack",
			request.MarshalJSON(ackData), request.JSONEncoding, a.cpHeaders())
		if err != nil {
			return err
		}

		err = a.identity.DoJSON(req, nil)
		if err == nil {
			return nil
		}
		// 422 is expected and indicates we should keep waiting.
		var se *request.StatusError
		if errors.As(err, &se) && se.HasStatus(http.StatusUnprocessableEntity) {
			continue
		}
		a.log.DEBUG.Printf("pollAck %s attempt %d/5 (ackId=%d): %v", action, i+1, ackID, err)
	}

	a.log.WARN.Printf("charger did not acknowledge %s within 5s, assuming it succeeded", action)

	return nil
}

// SetAmperageLimit sets the charge amperage limit on the given device via the
// internal REST API using PUT, as required by that endpoint.
func (a *API) SetAmperageLimit(deviceID int, limit int64) error {
	uri := fmt.Sprintf("%sapi/v1/configuration/chargers/%d/charge-amperage-limit", a.chargersURL, deviceID)

	data := struct {
		ChargeAmperageLimit int64 `json:"chargeAmperageLimit"`
	}{limit}

	return a.doJSON(func() (*http.Request, error) {
		return request.New(http.MethodPut, uri,
			request.MarshalJSON(data), request.JSONEncoding, a.cpInternalHeaders())
	}, nil)
}
