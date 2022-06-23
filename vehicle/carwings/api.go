package carwings

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type API struct {
	*request.Helper
	CustomSessionID string
	tz              string
	ResultKey       string
	RefreshTime     time.Time
}

func NewAPI(log *util.Logger, customSessionID, tz string) *API {
	return &API{
		Helper:          request.NewHelper(log),
		CustomSessionID: customSessionID,
		tz:              tz,
	}
}

func (v *API) Charger(vin string) (ChargerResponse, error) {
	// api result is stale
	if v.ResultKey != "" {
		if err := v.refreshResult(vin); err != nil {
			return ChargerResponse{}, err
		}
	}
	params := setCommonParams(vin, v.CustomSessionID, v.tz)

	uri := fmt.Sprint(BaseURL + "BatteryStatusRecordsRequest.php")

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(params), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"User-Agent":   "",
	})
	if err != nil {
		return ChargerResponse{}, err
	}

	var res ChargerResponse
	err = v.DoJSON(req, &res)
	if err != nil {
		return ChargerResponse{}, err
	}

	timestamp, err := time.Parse(time.RFC3339, res.BatteryStatusRecord.NotificationDateAndTime)
	if err != nil {
		return res, err
	}
	if elapsed := time.Since(timestamp); elapsed > carwingsStatusExpiry {
		if err = v.UpdateStatus(vin); err != nil {
			return ChargerResponse{}, err
		}
	} else {
		// reset if elapsed < carwingsStatusExpiry,
		// otherwise next check after soc timeout does not trigger update because refreshResult succeeds on old key
		v.ResultKey = ""
	}
	return res, nil
}

func (v *API) Climater(vin string) (ClimaterResponse, error) {
	params := setCommonParams(vin, v.CustomSessionID, v.tz)

	uri := fmt.Sprint(BaseURL + "RemoteACRecordsRequest.php")

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(params), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"User-Agent":   "",
	})
	if err != nil {
		return ClimaterResponse{}, err
	}

	var res ClimaterResponse
	err = v.DoJSON(req, &res)
	if err != nil {
		return ClimaterResponse{}, err
	}
	return res, nil
}

func setCommonParams(vin, customSessionID, tz string) url.Values {
	params := url.Values{}
	params.Set("RegionCode", "NE")
	params.Set("VIN", vin)
	params.Set("custom_sessionid", customSessionID)
	params.Set("tz", tz)

	return params
}

func (v *API) UpdateStatus(vin string) error {
	params := setCommonParams(vin, v.CustomSessionID, v.tz)

	uri := fmt.Sprint(BaseURL + "BatteryStatusCheckRequest.php")

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(params), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"User-Agent":   "",
	})
	if err != nil {
		return err
	}

	var res UpdateResponse
	err = v.DoJSON(req, &res)
	if err != nil {
		return err
	}
	v.ResultKey = res.ResultKey
	v.RefreshTime = time.Now()
	return nil
}

func (v *API) CheckUpdateStatus(vin, resultKey string) (bool, error) {
	params := setCommonParams(vin, v.CustomSessionID, v.tz)
	params.Set("resultKey", resultKey)

	uri := fmt.Sprint(BaseURL + "BatteryStatusCheckRequest.php")

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(params), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"User-Agent":   "",
	})
	if err != nil {
		return false, err
	}

	var res CheckUpdateResponse
	err = v.DoJSON(req, &res)
	if err != nil {
		return false, err
	}
	if res.OperationResult == "ELECTRIC_WAVE_ABNORMAL" {
		return false, errors.New("update failed")
	}
	return res.ResponseFlag == 1, nil
}

// refreshResult triggers an update if not already in progress, otherwise gets result
func (v *API) refreshResult(vin string) error {
	finished, err := v.CheckUpdateStatus(vin, v.ResultKey)

	// update successful and completed
	if err == nil && finished {
		v.ResultKey = ""
		return nil
	}

	// update still in progress, keep retrying
	if time.Since(v.RefreshTime) < carwingsRefreshTimeout {
		return api.ErrMustRetry
	}

	// give up
	v.ResultKey = ""
	if err == nil {
		err = api.ErrTimeout
	}

	return err
}
