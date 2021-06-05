package charger

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// HTTP testing appproach from http://hassansin.github.io/Unit-Testing-http-client-in-Go
type roundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// apiResponse helps to map an API Call to a test response
type apiResponse struct {
	apiCall     string
	apiResponse string
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(fn),
	}
}

// NewTestMobileConnect .
func NewTestMobileConnect(t *testing.T, responses []apiResponse) *MobileConnect {
	mcc := &MobileConnect{
		Helper:      request.NewHelper(util.NewLogger("foo")),
		uri:         "http://192.0.2.2:502",
		password:    "none",
		token:       "token",
		tokenExpiry: time.Now().Add(10 * time.Minute),
	}

	mcc.Client = NewTestClient(func(req *http.Request) *http.Response {
		// Each method may have multiple API calls, so we need to finde the proper
		// response string for the currently invoked call
		var responseString string
		for _, s := range responses {
			if strings.Contains("/"+string(s.apiCall), req.URL.Path) {
				responseString = s.apiResponse
			}
		}

		return &http.Response{
			StatusCode: 200,
			// Send response to be tested
			Body: io.NopCloser(bytes.NewBufferString(responseString)),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})

	return mcc
}

func TestMobileConnectLogin(t *testing.T) {
	tests := []struct {
		name      string
		responses []apiResponse
		password  string
		wantErr   bool
	}{
		// test cases for software version 2914
		{"login - success", []apiResponse{{mccAPILogin, "{\n    \"token\": \"1234567890._abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\"\n}"}}, "password", false},
		{"login - wrong password", []apiResponse{{mccAPILogin, "{\n    \"error\": \"wrong password\"\n}"}}, "wrong", true},
		{"login - bad return", []apiResponse{{mccAPILogin, "{{\n    \"error\": \"wrong password\"\n}"}}, "wrong", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			if err := mcc.login(tc.password); (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.login() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestMobileConnectRefresh(t *testing.T) {
	tests := []struct {
		name      string
		responses []apiResponse
		wantErr   bool
	}{
		// test cases for software version 2914
		{"refresh - success", []apiResponse{{mccAPIRefresh, "{\n    \"token\": \"1234567890._abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\"\n}"}}, false},
		{"refresh - wrong password", []apiResponse{{mccAPIRefresh, "{\n    \"error\": \"signature mismatch: OP-gWPOgQ9fdKujMgRNHkeH4WHqYrHe3Z2RqVXeUEuw1\"\n}"}}, true},
		{"refresh - bad return", []apiResponse{{mccAPIRefresh, "{{\n    \"error\": \"\"\n}"}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			if err := mcc.refresh(); (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.login() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestMobileConnectStatus(t *testing.T) {
	tests := []struct {
		name      string
		responses []apiResponse
		want      api.ChargeStatus
		wantErr   bool
	}{
		// test cases for software version 2914
		{"home plug - Unexpected API response", []apiResponse{{mccAPIChargeState, "abc"}}, api.StatusNone, true},
		{"home plug - Unplugged", []apiResponse{{mccAPIChargeState, "0\n"}}, api.StatusA, false},
		{"home plug - Connecting", []apiResponse{{mccAPIChargeState, "1\n"}}, api.StatusB, false},
		{"home plug - Error", []apiResponse{{mccAPIChargeState, "2\n"}}, api.StatusF, false},
		{"home plug - Established", []apiResponse{{mccAPIChargeState, "3\n"}}, api.StatusB, false},
		{"home plug - Paused", []apiResponse{{mccAPIChargeState, "4\n"}}, api.StatusB, false},
		{"home plug - Active", []apiResponse{{mccAPIChargeState, "5\n"}}, api.StatusC, false},
		{"home plug - Finished", []apiResponse{{mccAPIChargeState, "6\n"}}, api.StatusB, false},
		{"home plug - Unexpected status value", []apiResponse{{mccAPIChargeState, "10\n"}}, api.StatusNone, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			got, err := mcc.Status()
			if (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.Status() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("MobileConnect.Status() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMobileConnectEnabled(t *testing.T) {
	tests := []struct {
		name      string
		responses []apiResponse
		want      bool
		wantErr   bool
	}{
		// test cases for software version 2914
		{"home plug - Unexpected API response", []apiResponse{{mccAPIChargeState, "abc"}}, false, true},
		{"home plug - Unplugged", []apiResponse{{mccAPIChargeState, "0\n"}}, false, false},
		{"home plug - Connecting", []apiResponse{{mccAPIChargeState, "1\n"}}, false, false},
		{"home plug - Error", []apiResponse{{mccAPIChargeState, "2\n"}}, false, false},
		{"home plug - Established", []apiResponse{{mccAPIChargeState, "3\n"}}, false, false},
		{"home plug - Paused", []apiResponse{{mccAPIChargeState, "4\n"}}, true, false},
		{"home plug - Active", []apiResponse{{mccAPIChargeState, "5\n"}}, true, false},
		{"home plug - Finished", []apiResponse{{mccAPIChargeState, "6\n"}}, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			got, err := mcc.Enabled()
			if (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.Enabled() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("MobileConnect.Enabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMobileConnectMaxCurrent(t *testing.T) {
	tests := []struct {
		name      string
		responses []apiResponse
		current   int64
		wantErr   bool
	}{
		// test cases for software version 2914
		{
			"home plug - success min value",
			[]apiResponse{
				{mccAPICurrentCableInformation, "\"{\\\"carCable\\\":5,\\\"gridCable\\\":8,\\\"hwfpMaxLimit\\\":32,\\\"maxValue\\\":10,\\\"minValue\\\":6,\\\"value\\\":10}\""},
				{mccAPISetCurrentLimit, "\"OK\"\n"},
			},
			6, false,
		},
		{
			"home plug - success max value",
			[]apiResponse{
				{mccAPICurrentCableInformation, "\"{\\\"carCable\\\":5,\\\"gridCable\\\":8,\\\"hwfpMaxLimit\\\":32,\\\"maxValue\\\":10,\\\"minValue\\\":6,\\\"value\\\":10}\""},
				{mccAPISetCurrentLimit, "\"OK\"\n"},
			},
			10, false,
		},
		{
			"home plug - error value too small",
			[]apiResponse{
				{mccAPICurrentCableInformation, "\"{\\\"carCable\\\":5,\\\"gridCable\\\":8,\\\"hwfpMaxLimit\\\":32,\\\"maxValue\\\":10,\\\"minValue\\\":6,\\\"value\\\":10}\""},
			},
			0, true,
		},
		{
			"home plug - error value too big",
			[]apiResponse{
				{mccAPICurrentCableInformation, "\"{\\\"carCable\\\":5,\\\"gridCable\\\":8,\\\"hwfpMaxLimit\\\":32,\\\"maxValue\\\":10,\\\"minValue\\\":6,\\\"value\\\":10}\""},
			},
			16, true,
		},
		{
			"home plug - 1st API success but 2nd API error",
			[]apiResponse{
				{mccAPICurrentCableInformation, "\"{\\\"carCable\\\":5,\\\"gridCable\\\":8,\\\"hwfpMaxLimit\\\":32,\\\"maxValue\\\":10,\\\"minValue\\\":6,\\\"value\\\":10}\""},
				{mccAPISetCurrentLimit, "Unexpected response"},
			},
			10, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			if err := mcc.MaxCurrent(tc.current); (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.MaxCurrent() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestMobileConnectCurrentPower(t *testing.T) {
	tests := []struct {
		name      string
		responses []apiResponse
		want      float64
		wantErr   bool
	}{
		// test cases for software version 2914
		{
			"home plug - charging",
			[]apiResponse{
				{mccAPIEnergy, "\"{\\n    \\\"L1\\\": {\\n        \\\"Ampere\\\": 9.9000000000000004,\\n        \\\"Power\\\": 2308,\\n        \\\"Volts\\\": 230.5\\n    },\\n    \\\"L2\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 13.700000000000001\\n    },\\n    \\\"L3\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 13.9\\n    }\\n}\\n\""},
			},
			2308, false,
		},
		{
			"3 phase low power - charging",
			[]apiResponse{
				{mccAPIEnergy, "\"{\\n    \\\"L1\\\": {\\n        \\\"Ampere\\\": 0.5,\\n        \\\"Power\\\": 7,\\n        \\\"Volts\\\": 244.40000000000001\\n    },\\n    \\\"L2\\\": {\\n        \\\"Ampere\\\": 0.5,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 242.10000000000002\\n    },\\n    \\\"L3\\\": {\\n        \\\"Ampere\\\": 0.5,\\n        \\\"Power\\\": 1,\\n        \\\"Volts\\\": 242.30000000000001\\n    }\\n}\\n\""},
			},
			8, false,
		},
		{
			"no data response",
			[]apiResponse{
				{mccAPIEnergy, "\"\"\n"},
			}, 0, false,
		},
		{
			"home plug - error response",
			[]apiResponse{
				{mccAPIEnergy, "\"{\\n    \\\"L1\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 246.60000000000002\\n    },\\n    \\\"L2\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 16.800000000000001\\n    },\\n    \\\"L3\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 16.300000000000001\\n    }\\n}\\n\""},
			}, 0, false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			got, err := mcc.CurrentPower()
			if (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.CurrentPower() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("MobileConnect.CurrentPower() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMobileConnectChargedEnergy(t *testing.T) {
	tests := []struct {
		name      string
		responses []apiResponse
		want      float64
		wantErr   bool
	}{
		// test cases for software version 2914
		{
			"valid response",
			[]apiResponse{
				{mccAPICurrentSession, "\"{\\n    \\\"account\\\": \\\"PRIVATE\\\",\\n    \\\"chargingRate\\\": 0,\\n    \\\"chargingType\\\": \\\"AC\\\",\\n    \\\"clockSrc\\\": \\\"NTP\\\",\\n    \\\"costs\\\": 0,\\n    \\\"currency\\\": \\\"\\\",\\n    \\\"departTime\\\": \\\"\\\",\\n    \\\"duration\\\": 30789,\\n    \\\"endOfChargeTime\\\": \\\"\\\",\\n    \\\"endSoc\\\": 0,\\n    \\\"endTime\\\": \\\"\\\",\\n    \\\"energySumKwh\\\": 18.832000000000001,\\n    \\\"evChargingRatekW\\\": 0,\\n    \\\"evTargetSoc\\\": -1,\\n    \\\"evVasAvailability\\\": false,\\n    \\\"pcid\\\": \\\"\\\",\\n    \\\"powerRange\\\": 0,\\n    \\\"selfEnergy\\\": 0,\\n    \\\"sessionId\\\": 13,\\n    \\\"soc\\\": -1,\\n    \\\"solarEnergyShare\\\": 0,\\n    \\\"startSoc\\\": 0,\\n    \\\"startTime\\\": \\\"2020-04-15T10:07:22+02:00\\\",\\n    \\\"totalRange\\\": 0,\\n    \\\"vehicleBrand\\\": \\\"\\\",\\n    \\\"vehicleModel\\\": \\\"\\\",\\n    \\\"whitelist\\\": false\\n}\\n\""},
			}, 18.832000000000001, false,
		},
		{
			"no data response",
			[]apiResponse{
				{mccAPICurrentSession, "\"\"\n"},
			}, 0, false,
		},
		{
			"error response",
			[]apiResponse{
				{mccAPICurrentSession, "invalidjson"},
			}, 0, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			got, err := mcc.ChargedEnergy()
			if (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.ChargedEnergy() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("MobileConnect.ChargedEnergy() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMobileConnectChargingTime(t *testing.T) {
	tests := []struct {
		name      string
		responses []apiResponse
		want      time.Duration
		wantErr   bool
	}{
		{
			"valid response",
			[]apiResponse{
				{mccAPICurrentSession, "\"{\\n    \\\"account\\\": \\\"PRIVATE\\\",\\n    \\\"chargingRate\\\": 0,\\n    \\\"chargingType\\\": \\\"AC\\\",\\n    \\\"clockSrc\\\": \\\"NTP\\\",\\n    \\\"costs\\\": 0,\\n    \\\"currency\\\": \\\"\\\",\\n    \\\"departTime\\\": \\\"\\\",\\n    \\\"duration\\\": 30789,\\n    \\\"endOfChargeTime\\\": \\\"\\\",\\n    \\\"endSoc\\\": 0,\\n    \\\"endTime\\\": \\\"\\\",\\n    \\\"energySumKwh\\\": 18.832000000000001,\\n    \\\"evChargingRatekW\\\": 0,\\n    \\\"evTargetSoc\\\": -1,\\n    \\\"evVasAvailability\\\": false,\\n    \\\"pcid\\\": \\\"\\\",\\n    \\\"powerRange\\\": 0,\\n    \\\"selfEnergy\\\": 0,\\n    \\\"sessionId\\\": 13,\\n    \\\"soc\\\": -1,\\n    \\\"solarEnergyShare\\\": 0,\\n    \\\"startSoc\\\": 0,\\n    \\\"startTime\\\": \\\"2020-04-15T10:07:22+02:00\\\",\\n    \\\"totalRange\\\": 0,\\n    \\\"vehicleBrand\\\": \\\"\\\",\\n    \\\"vehicleModel\\\": \\\"\\\",\\n    \\\"whitelist\\\": false\\n}\\n\""},
			}, 30789 * time.Second, false,
		},
		{
			"no data response",
			[]apiResponse{
				{mccAPICurrentSession, "\"\"\n"},
			}, 0, false,
		},
		{
			"error response",
			[]apiResponse{
				{mccAPICurrentSession, "invalidjson"},
			}, 0, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			got, err := mcc.ChargingTime()
			if (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.ChargingTime() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("MobileConnect.ChargingTime() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMobileConnectCurrents(t *testing.T) {
	tests := []struct {
		name                   string
		responses              []apiResponse
		wantL1, wantL2, wantL3 float64
		wantErr                bool
	}{
		// test cases for software version 2914
		{
			"home plug - charging",
			[]apiResponse{
				{mccAPIEnergy, "\"{\\n    \\\"L1\\\": {\\n        \\\"Ampere\\\": 9.9000000000000004,\\n        \\\"Power\\\": 2308,\\n        \\\"Volts\\\": 230.5\\n    },\\n    \\\"L2\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 13.700000000000001\\n    },\\n    \\\"L3\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 13.9\\n    }\\n}\\n\""},
			},
			9.9000000000000004, 0, 0, false,
		},
		{
			"3 phase low power - charging",
			[]apiResponse{
				{mccAPIEnergy, "\"{\\n    \\\"L1\\\": {\\n        \\\"Ampere\\\": 0.5,\\n        \\\"Power\\\": 7,\\n        \\\"Volts\\\": 244.40000000000001\\n    },\\n    \\\"L2\\\": {\\n        \\\"Ampere\\\": 0.5,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 242.10000000000002\\n    },\\n    \\\"L3\\\": {\\n        \\\"Ampere\\\": 0.5,\\n        \\\"Power\\\": 1,\\n        \\\"Volts\\\": 242.30000000000001\\n    }\\n}\\n\""},
			},
			0.5, 0.5, 0.5, false,
		},
		{
			"no data response",
			[]apiResponse{
				{mccAPIEnergy, "\"\"\n"},
			}, 0, 0, 0, false,
		},
		{
			"home plug - error response",
			[]apiResponse{
				{mccAPIEnergy, "\"{\\n    \\\"L1\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 246.60000000000002\\n    },\\n    \\\"L2\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 16.800000000000001\\n    },\\n    \\\"L3\\\": {\\n        \\\"Ampere\\\": 0,\\n        \\\"Power\\\": 0,\\n        \\\"Volts\\\": 16.300000000000001\\n    }\\n}\\n\""},
			}, 0, 0, 0, false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcc := NewTestMobileConnect(t, tc.responses)

			gotL1, gotL2, gotL3, err := mcc.Currents()
			if (err != nil) != tc.wantErr {
				t.Errorf("MobileConnect.Currents() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if gotL1 != tc.wantL1 {
				t.Errorf("MobileConnect.Currents() = %v, want %v", gotL1, tc.wantL1)
			}
			if gotL2 != tc.wantL2 {
				t.Errorf("MobileConnect.Currents() = %v, want %v", gotL2, tc.wantL2)
			}
			if gotL3 != tc.wantL3 {
				t.Errorf("MobileConnect.Currents() = %v, want %v", gotL3, tc.wantL3)
			}
		})
	}
}
