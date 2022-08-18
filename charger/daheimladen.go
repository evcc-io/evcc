package charger

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/daheimladen"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// DaheimLaden charger implementation
type DaheimLaden struct {
	*request.Helper
	stationID     string
	connectorID   int32
	idTag         string
	token         string
	transactionID int32
}

func init() {
	registry.Add("daheimladen", NewDaheimLadenFromConfig)
}

// NewDaheimLadenFromConfig creates a DaheimLaden charger from generic config
func NewDaheimLadenFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		Token     string
		StationID string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDaheimLaden(cc.Token, cc.StationID)
}

// NewDaheimLaden creates DaheimLaden charger
func NewDaheimLaden(token, stationID string) (*DaheimLaden, error) {
	c := &DaheimLaden{
		Helper:      request.NewHelper(util.NewLogger("daheim")),
		stationID:   stationID,
		connectorID: 1,
		idTag:       daheimladen.EVCC_IDTAG,
		token:       token,
	}

	c.Client.Transport = &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
			TokenType:   "Bearer",
		}),
		Base: c.Client.Transport,
	}
	return c, nil
}

// Enabled implements the api.Charger interface
func (c *DaheimLaden) Enabled() (bool, error) {
	var res daheimladen.GetLatestStatus
	err := c.GetJSON(fmt.Sprintf("%s/cs/%s/status", daheimladen.BASE_URL, c.stationID), &res)
	return res.Status == string(daheimladen.CHARGING), err
}

// Enable implements the api.Charger interface
func (c *DaheimLaden) Enable(enable bool) error {
	if enable {
		data := daheimladen.RemoteStartRequest{
			ConnectorID: c.connectorID,
			IdTag:       c.idTag,
		}

		uri := fmt.Sprintf("%s/cs/%s/remotestart", daheimladen.BASE_URL, c.stationID)
		req, err := http.NewRequest(http.MethodPost, uri, request.MarshalJSON(data))
		if err != nil {
			return err
		}

		var res daheimladen.RemoteStartResponse
		if err = c.DoJSON(req, &res); err == nil && res.Status != string(daheimladen.REMOTE_START_ACCEPTED) {
			err = fmt.Errorf("charging station refused to start transaction")
		}

		return err
	}

	var res daheimladen.GetLatestInProgressTransactionResponse
	uri := fmt.Sprintf("%s/cs/%s/get_latest_inprogress_transaction", daheimladen.BASE_URL, c.stationID)
	if err := c.GetJSON(uri, &res); err != nil {
		return err
	}

	c.transactionID = res.TransactionID

	data := daheimladen.RemoteStopRequest{
		TransactionID: c.transactionID,
	}

	uri = fmt.Sprintf("%s/cs/%s/remotestop", daheimladen.BASE_URL, c.stationID)
	req, err := http.NewRequest(http.MethodPost, uri, request.MarshalJSON(data))
	if err != nil {
		return err
	}

	var remoteStopRes daheimladen.RemoteStartResponse
	if err = c.DoJSON(req, &remoteStopRes); err == nil && remoteStopRes.Status != string(daheimladen.REMOTE_STOP_ACCEPTED) {
		err = fmt.Errorf("charging station refused to stop transaction")
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *DaheimLaden) MaxCurrent(current int64) error {
	data := daheimladen.ChangeConfigurationRequest{
		Key:   string(daheimladen.CHARGE_RATE),
		Value: strconv.FormatInt(current, 10),
	}

	uri := fmt.Sprintf("%s/cs/%s/change_config", daheimladen.BASE_URL, c.stationID)
	req, err := http.NewRequest(http.MethodPost, uri, request.MarshalJSON(data))
	if err != nil {
		return err
	}

	var res daheimladen.ChangeConfigurationResponse
	if err = c.DoJSON(req, &res); err == nil && res.Status != string(daheimladen.CHANGE_CONFIG_ACCEPTED) {
		err = fmt.Errorf("charging station refused to change max current")
	}

	return err
}

// Status implements the api.Charger interface
func (c *DaheimLaden) Status() (api.ChargeStatus, error) {
	var res daheimladen.GetLatestStatus
	uri := fmt.Sprintf("%s/cs/%s/status", daheimladen.BASE_URL, c.stationID)
	if err := c.GetJSON(uri, &res); err != nil {
		return api.StatusNone, err
	}

	status := daheimladen.ChargePointStatus(res.Status)
	switch status {
	case daheimladen.AVAILABLE:
		return api.StatusA, nil
	case daheimladen.PREPARING:
		return api.StatusB, nil
	case daheimladen.CHARGING, daheimladen.FINISHING:
		return api.StatusC, nil
	case daheimladen.FAULTED:
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", res.Status)
	}
}

var _ api.Meter = (*DaheimLaden)(nil)

// CurrentPower implements the api.Meter interface
func (c *DaheimLaden) CurrentPower() (float64, error) {
	var res daheimladen.GetLatestMeterValueResponse
	uri := fmt.Sprintf("%s/cs/%s/metervalue", daheimladen.BASE_URL, c.stationID)
	err := c.GetJSON(uri, &res)
	return float64(res.ActivePowerImport * 1e3), err
}

var _ api.MeterEnergy = (*DaheimLaden)(nil)

// TotalEnergy implements the api.MeterMeterEnergy interface
func (c *DaheimLaden) TotalEnergy() (float64, error) {
	var res daheimladen.GetLatestMeterValueResponse
	uri := fmt.Sprintf("%s/cs/%s/metervalue", daheimladen.BASE_URL, c.stationID)
	err := c.GetJSON(uri, &res)
	return float64(res.EnergyActiveImportRegister), err
}

var _ api.MeterCurrent = (*DaheimLaden)(nil)

// Currents implements the api.MeterCurrent interface
func (c *DaheimLaden) Currents() (float64, float64, float64, error) {
	var res daheimladen.GetLatestMeterValueResponse
	uri := fmt.Sprintf("%s/cs/%s/metervalue", daheimladen.BASE_URL, c.stationID)
	err := c.GetJSON(uri, &res)
	return float64(res.CurrentImportPhaseL1), float64(res.CurrentImportPhaseL2), float64(res.CurrentImportPhaseL3), err
}
