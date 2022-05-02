package charger

// LICENSE

// Copyright (c) 2019-2022 andig

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/smaevcharger"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"golang.org/x/oauth2"
)

// smaevchager charger implementation
type Smaevcharger struct {
	*request.Helper
	log              *util.Logger
	host             string // 192.168.XXX.XXX
	user             string // LOGIN user
	password         string // password
	MeasurementsData []smaevcharger.Measurements
	ParametersData   []smaevcharger.Parameters
}

func init() {
	registry.Add("smaevcharger", NewSmaevchargerFromConfig)
}

// NewSmaevchargerFromConfig creates a Smaevcharger charger from generic config
func NewSmaevchargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Host     string
		User     string
		Password string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Host == "" {
		return nil, errors.New("missing host")
	}
	if cc.User == "" {
		return nil, errors.New("missing user")
	} else if cc.User == "admin" {
		return nil, errors.New("user admin not allowed, create new user")
	}
	if cc.Password == "" {
		return nil, errors.New("missing password")
	}

	return NewSmaevcharger(cc.Host, cc.User, cc.Password)
}

// NewSmaevcharger creates Smaevcharger charger
func NewSmaevcharger(host string, user string, password string) (api.Charger, error) {
	log := util.NewLogger("smaevcharger").Redact(user, password)

	wb := &Smaevcharger{
		Helper:   request.NewHelper(log),
		log:      log,
		host:     "http://" + host + "/api/v1",
		user:     user,
		password: password,
	}

	ts, err := smaevcharger.TokenSource(log, wb.host, wb.user, wb.password)
	if err != nil {
		return wb, err
	}

	// replace client transport with authenticated transport
	wb.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   wb.Client.Transport,
	}

	SoftwareVersion := fmt.Sprint(wb.GetParameter("Parameter.Nameplate.PkgRev"))
	SoftwareVersionParts := strings.Split(SoftwareVersion, ".")
	errortext := "Failed to read Charger Softwareversion"

	if len(SoftwareVersionParts) < 3 {
		return wb, errors.New(errortext)
	}
	tempvar1, err := strconv.Atoi(SoftwareVersionParts[0])
	if err != nil {
		return wb, errors.New(errortext)
	}
	tempvar2, err := strconv.Atoi(SoftwareVersionParts[1])
	if err != nil {
		return wb, errors.New(errortext)
	}
	tempvar3, err := strconv.Atoi(SoftwareVersionParts[2])
	if err != nil {
		return wb, errors.New(errortext)
	}

	if tempvar1 < 1 || tempvar2 < 2 || tempvar3 < 23 {
		return wb, errors.New("Charger Softwareversion not supported - please update > 1.2.23R")
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Smaevcharger) Status() (api.ChargeStatus, error) {
	StateChargerCharging := wb.GetMeasurement("Measurement.Operation.EVeh.ChaStt")

	switch StateChargerCharging {
	case smaevcharger.ConstNConNCarNChar: // No Car connectec and no charging
		return api.StatusA, nil
	case smaevcharger.ConstYConYCarNChar: // Car connected and no charging
		return api.StatusB, nil
	case smaevcharger.ConstYConYCarYChar: // Car connected and charging
		return api.StatusC, nil
	}
	return api.StatusNone, fmt.Errorf("SMA EV Charger state: %s", StateChargerCharging)
}

// Enabled implements the api.Charger interface
func (wb *Smaevcharger) Enabled() (bool, error) {
	StateChargerMode := wb.GetParameter("Parameter.Chrg.ActChaMod")

	switch StateChargerMode {
	case smaevcharger.ConstFastCharge: // Schnellladen - 4718
		return true, nil
	case smaevcharger.ConstOptiCharge: // Optimiertes Laden - 4719
		return true, nil
	case smaevcharger.ConstPlanCharge: // Laden mit Vorgabe - 4720
		return true, nil
	case smaevcharger.ConstStopCharge: // Ladestopp - 4721
		return false, nil
	}
	return false, fmt.Errorf("SMA EV Charger  charge mode: %s", StateChargerMode)
}

// Enable implements the api.Charger interface
func (wb *Smaevcharger) Enable(enable bool) error {
	StateChargerSwitch := wb.GetMeasurement("Measurement.Chrg.ModSw")
	if enable {
		switch StateChargerSwitch {
		case smaevcharger.ConstSwitchOeko: // Switch PV Loading
			wb.SendParameter("Parameter.Chrg.ActChaMod", smaevcharger.ConstOptiCharge)
		case smaevcharger.ConstSwitchFast: // Fast charging
			wb.SendParameter("Parameter.Chrg.ActChaMod", smaevcharger.ConstFastCharge)
		}
	} else {
		wb.SendParameter("Parameter.Chrg.ActChaMod", smaevcharger.ConstStopCharge)
	}
	time.Sleep(time.Second) //Some Delay to prevent out of Sync - The Charger needs some time to react after setting have been changed
	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Smaevcharger) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %v", current)
	}

	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Smaevcharger)(nil)

// maxCurrentMillis implements the api.ChargerEx interface
func (wb *Smaevcharger) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}
	wb.SendParameter("Parameter.Inverter.AcALim", strconv.FormatFloat(current, 'f', 2, 64))
	time.Sleep(time.Second)
	return nil
}

var _ api.Meter = (*Smaevcharger)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Smaevcharger) CurrentPower() (float64, error) {
	var Power = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.Metering.GridMs.TotWIn"))
	return Power, nil
}

var _ api.ChargeRater = (*Smaevcharger)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Smaevcharger) ChargedEnergy() (float64, error) {
	var data = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.ChaSess.WhIn"))
	return float64(data / 1000.0), nil
}

var _ api.MeterCurrent = (*Smaevcharger)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Smaevcharger) Currents() (float64, float64, float64, error) {
	var PhsA = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.GridMs.A.phsA"))
	var PhsB = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.GridMs.A.phsB"))
	var PhsC = wb.ConvertInterfaceToFloat(wb.GetMeasurement("Measurement.GridMs.A.phsC"))
	return PhsA, PhsB, PhsC, nil
}

func (wb *Smaevcharger) GetMeasurementData() bool {
	Host := wb.host + "/measurements/live"
	jsonData := []byte(`[{"componentId": "IGULD:SELF"}]`)

	req, err := request.New(http.MethodPost, Host, bytes.NewBuffer(jsonData), request.JSONEncoding)
	if err == nil {
		err = wb.DoJSON(req, &wb.MeasurementsData)
		return err == nil
	}
	return false
}

func (wb *Smaevcharger) GetParameterData() bool {
	Host := wb.host + "/parameters/search/"
	jsonData := []byte(`{"queryItems":[{"componentId":"IGULD:SELF"}]}`)

	req, err := request.New(http.MethodPost, Host, bytes.NewBuffer(jsonData), request.JSONEncoding)
	if err == nil {
		err = wb.DoJSON(req, &wb.ParametersData)
		return err == nil
	}
	return false
}

func (wb *Smaevcharger) GetMeasurement(id string) interface{} {
	dataok := wb.GetMeasurementData()
	if !dataok {
		nil := fmt.Errorf("failed to aquire measurement data")
		return nil
	}
	var returndata interface{}

	for i := range wb.MeasurementsData {
		if wb.MeasurementsData[i].ChannelId == id {
			returndata = wb.MeasurementsData[i].Values[0].Value
			return returndata
		}
	}
	return returndata
}

func (wb *Smaevcharger) GetParameter(id string) interface{} {
	dataok := wb.GetParameterData()
	if !dataok {
		nil := fmt.Errorf("failed to aquire parameter data")
		return nil
	}
	var returndata interface{}

	for i := range wb.ParametersData[0].Values {
		if wb.ParametersData[0].Values[i].ChannelId == id {
			returndata = wb.ParametersData[0].Values[i].Value
			return returndata
		}
	}
	return returndata
}

func (wb *Smaevcharger) SendParameter(id string, value string) bool {
	if wb.ParametersData == nil {
		wb.GetParameterData()
	}
	var parameter smaevcharger.SendParameter
	var data smaevcharger.SendData

	data.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	data.ChannelId = id
	data.Value = value
	parameter.Values = append(parameter.Values, data)

	parameterAvaliable := false
	for i := range wb.ParametersData[0].Values {
		if wb.ParametersData[0].Values[i].ChannelId == id {
			parameterAvaliable = true
			break
		}
	}

	if parameterAvaliable {

		Host := wb.host + "/parameters/IGULD:SELF/"
		jsonData, err := json.Marshal(parameter)
		if err != nil {
			return false
		}
		req, err := request.New(http.MethodPut, Host, bytes.NewBuffer(jsonData), request.JSONEncoding)
		if err == nil {
			resp, err := wb.Do(req)
			if resp.StatusCode >= 200 && resp.StatusCode <= 299 && err == nil {
				return true
			}
		}
	}
	return false
}

func (wb *Smaevcharger) SendMultiParameter(data []smaevcharger.SendData) bool {
	if wb.ParametersData == nil {
		wb.GetParameterData()
	}
	var parameter smaevcharger.SendParameter
	var payload smaevcharger.SendData

	for i := range data {
		payload.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		payload.ChannelId = data[i].ChannelId
		payload.Value = data[i].Value
		parameter.Values = append(parameter.Values, payload)
	}

	Host := wb.host + "/parameters/IGULD:SELF/"
	jsonData, err := json.Marshal(parameter)
	if err != nil {
		return false
	}
	req, err := request.New(http.MethodPut, Host, bytes.NewBuffer(jsonData), request.JSONEncoding)
	if err == nil {
		resp, err := wb.Do(req)
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 && err == nil {
			return true
		}
	}
	return false
}

func (wb *Smaevcharger) ConvertInterfaceToFloat(data interface{}) float64 {
	var dataout float64
	switch value := data.(type) {
	case float32:
		dataout = float64(value)
	}
	return dataout
}
