package vehicle

import (
	"encoding/json"
	"fmt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestKamereonVehicles(t *testing.T) {
	testTable := []struct {
		name             string
		configVIN        string
		server           *httptest.Server
		expectedResponse []string
		expectedErr      error
	}{
		{name: "kamereon-response-with-role-set-vin-set-in-config",
			configVIN: "V1234",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				kamereonResponse := createResponseStub("ACTIVE", "some role")
				kamereonResponseJson, _ := json.Marshal(kamereonResponse)
				w.WriteHeader(http.StatusOK)
				w.Write(kamereonResponseJson)
			})), expectedResponse: []string{"V1234"}, expectedErr: nil},
		{name: "kamereon-response-with-role-set-no-vin-in-config",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				kamereonResponse := createResponseStub("ACTIVE", "some role")
				kamereonResponseJson, _ := json.Marshal(kamereonResponse)
				w.WriteHeader(http.StatusOK)
				w.Write(kamereonResponseJson)
			})), expectedResponse: []string{"V1234"}, expectedErr: nil},
		{name: "kamereon-response-with-role-not-set",
			configVIN: "V1234",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				kamereonResponse := createResponseStub("ACTIVE", "")
				kamereonResponseJson, _ := json.Marshal(kamereonResponse)
				w.WriteHeader(http.StatusOK)
				w.Write(kamereonResponseJson)
			})), expectedResponse: nil, expectedErr: fmt.Errorf("No paired vehicle found. %s %s",
				"For the configured vehicle with vin: V1234 the connected driver role is not set.",
				" Renault will reject all car status requests with a http 403 error code. "+
					" Please pair your configured vehicle with the used my-renault account."),
		},
		{name: "kamereon-response-with-status-invalid",
			configVIN: "V1234",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				kamereonResponse := createResponseStub("INACTIVE", "some role")
				kamereonResponseJson, _ := json.Marshal(kamereonResponse)
				w.WriteHeader(http.StatusOK)
				w.Write(kamereonResponseJson)
			})), expectedResponse: nil, expectedErr: fmt.Errorf("No paired vehicle found. %s %s",
				"For the configured vehicle with vin: V1234 the my-renault status is not set to active.",
				" Renault will reject all car status requests with a http 403 error code. "+
					" Please pair your configured vehicle with the used my-renault account."),
		},
	}
	for _, tc := range testTable {
		t.Run(tc.name, func(t *testing.T) {
			defer tc.server.Close()
			resp, err := runTestKamereonVehicles(tc.server.URL, tc.configVIN)
			if !reflect.DeepEqual(resp, tc.expectedResponse) {
				t.Errorf("\nexpected (%v)\n got     (%v)", tc.expectedResponse, resp)
			}
			if !reflect.DeepEqual(err, tc.expectedErr) {
				t.Errorf("\nexpected (%v)\n got     (%v)", tc.expectedErr, err)
			}
		})
	}
}

func createResponseStub(status string, role string) *kamereonResponse {
	kamereonResponse := &kamereonResponse{
		Accounts:    []kamereonAccount{{AccountID: "1"}},
		AccessToken: "access-token",
		VehicleLinks: []kamereonVehicle{{VIN: "V1234", Brand: "renault", Status: status,
			ConnectedDriver: connectedDriver{Role: role}}}}
	return kamereonResponse
}

func runTestKamereonVehicles(serverUri string, configVIN string) ([]string, error) {
	mockUser := "mockUser"
	mockPassword := "mockPassword"
	log := util.NewLogger("renault").Redact(mockUser, mockPassword)

	v := &Renault{
		Helper:   request.NewHelper(log),
		user:     mockUser,
		password: mockPassword,
	}
	v.gigya = configServer{serverUri + "/gigya", "mock-gigya-api-key"}
	v.kamereon = configServer{serverUri + "/kamereon", "mock-kamereon-api-key"}
	return v.kamereonVehicles(configVIN)
}
