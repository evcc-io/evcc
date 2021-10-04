package charger

import (
	"fmt"
	"net/http"

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
	cc := struct {
		Token             string
		ChargingStationID string
		IDTag             string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDaheimLaden(cc.Token, cc.ChargingStationID, cc.IDTag)
}

// NewDaheimLaden creates DaheimLaden charger
func NewDaheimLaden(token string, stationID string, idTag string) (*DaheimLaden, error) {
	c := &DaheimLaden{
		Helper:      request.NewHelper(util.NewLogger("daheim")),
		stationID:   stationID,
		connectorID: 1,
		idTag:       idTag,
		token:       token,
	}
	c.Timeout = 0
	c.Client.Transport = &oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
			TokenType:   "Bearer",
		}),
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *DaheimLaden) Enabled() (bool, error) {
	var res daheimladen.GetLatestStatus
	err := c.GetJSON(fmt.Sprintf("%s/cs/%s/status", daheimladen.BASE_URL, c.stationID), &res)
	if err != nil {
		return false, err
	}

	if res.Status == string(daheimladen.CHARGING) || res.Status == string(daheimladen.PREPARING) {
		return true, nil
	}
	return false, nil
}

// Enable implements the api.Charger interface
func (c *DaheimLaden) Enable(enable bool) error {
	if enable {
		remoteStartReq := daheimladen.RemoteStartRequest{
			ConnectorID: c.connectorID,
			IdTag:       c.idTag,
		}
		data := request.MarshalJSON(remoteStartReq)
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/cs/%s/remotestart", daheimladen.BASE_URL, c.stationID), data)
		if err != nil {
			return err
		}
		var remoteStartRes daheimladen.RemoteStartResponse
		err = c.DoJSON(req, &remoteStartRes)
		if err != nil {
			return err
		}
		if remoteStartRes.Status != string(daheimladen.REMOTE_START_ACCEPTED) {
			return fmt.Errorf("charging station refused to start transaction")
		}
		return nil
	}

	var latestTransactionRes daheimladen.GetLatestInProgressTransactionResponse
	err := c.GetJSON(fmt.Sprintf("%s/cs/%s/get_latest_inprogress_transaction", daheimladen.BASE_URL, c.stationID), &latestTransactionRes)
	if err != nil {
		return err
	}

	c.transactionID = latestTransactionRes.TransactionID

	if c.transactionID == 0 {
		return fmt.Errorf("cannot stop transaction as the transaction was started with plug and charge mode")
	}
	remoteStopReq := daheimladen.RemoteStopRequest{
		TransactionID: c.transactionID,
	}
	data := request.MarshalJSON(remoteStopReq)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/cs/%s/remotestop", daheimladen.BASE_URL, c.stationID), data)
	if err != nil {
		return err
	}
	var remoteStopRes daheimladen.RemoteStartResponse
	err = c.DoJSON(req, &remoteStopRes)
	if err != nil {
		return err
	}
	if remoteStopRes.Status != string(daheimladen.REMOTE_STOP_ACCEPTED) {
		return fmt.Errorf("charging station refused to stop transaction")
	}
	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *DaheimLaden) MaxCurrent(current int64) error {
	changeConfigReq := daheimladen.ChangeConfigurationRequest{
		Key:   string(daheimladen.CHARGE_RATE),
		Value: fmt.Sprint(current),
	}

	data := request.MarshalJSON(changeConfigReq)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/cs/%s/change_config", daheimladen.BASE_URL, c.stationID), data)
	if err != nil {
		return err
	}
	var res daheimladen.ChangeConfigurationResponse
	err = c.DoJSON(req, &res)
	if err != nil {
		return err
	}

	if res.Status != string(daheimladen.CHANGE_CONFIG_ACCEPTED) {
		return fmt.Errorf("charging station refused to change max current")
	}
	return nil
}

// Status implements the api.Charger interface
func (c *DaheimLaden) Status() (api.ChargeStatus, error) {
	var res daheimladen.GetLatestStatus
	err := c.GetJSON(fmt.Sprintf("%s/cs/%s/status", daheimladen.BASE_URL, c.stationID), &res)
	if err != nil {
		return api.StatusNone, err
	}

	status := daheimladen.ChargePointStatus(res.Status)
	switch status {
	case daheimladen.AVAILABLE:
		return api.StatusA, nil
	case daheimladen.PREPARING:
		return api.StatusB, nil
	case daheimladen.CHARGING:
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
	err := c.GetJSON(fmt.Sprintf("%s/cs/%s/metervalue", daheimladen.BASE_URL, c.stationID), &res)
	if err != nil {
		return float64(0), err
	}
	return float64(res.PowerActiveImport), nil
}

var _ api.MeterEnergy = (*DaheimLaden)(nil)

// TotalEnergy implements the api.MeterMeterEnergy interface
func (c *DaheimLaden) TotalEnergy() (float64, error) {
	var res daheimladen.GetLatestMeterValueResponse
	err := c.GetJSON(fmt.Sprintf("%s/cs/%s/metervalue", daheimladen.BASE_URL, c.stationID), &res)
	if err != nil {
		return float64(0), err
	}
	return float64(res.EnergyActiveImportRegister) / float64(1000), nil
}

var _ api.MeterCurrent = (*DaheimLaden)(nil)

// Currents implements the api.MeterCurrent interface
func (c *DaheimLaden) Currents() (float64, float64, float64, error) {
	var res daheimladen.GetLatestMeterValueResponse
	err := c.GetJSON(fmt.Sprintf("%s/cs/%s/metervalue", daheimladen.BASE_URL, c.stationID), &res)
	if err != nil {
		return float64(0), float64(0), float64(0), err
	}
	return float64(res.CurrentImportPhaseL1), float64(res.CurrentImportPhaseL2), float64(res.CurrentImportPhaseL3), nil
}
