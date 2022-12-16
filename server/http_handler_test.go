package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

type mockLoadpoint struct {
	SoC        int
	TargetTime time.Time
}

func (lp *mockLoadpoint) SetTargetCharge(time time.Time, soc int) error {
	lp.SoC = soc
	lp.TargetTime = time
	return nil
}

func TestTargetChargeHandler(t *testing.T) {
	tc := []struct {
		inSoC      string
		inTime     string
		statusCode int
		outSoC     int
		outTime    time.Time
	}{
		{
			"70", "2022-05-17T06:20:59.509Z", http.StatusOK,
			70, time.Date(2022, 0o5, 17, 0o6, 20, 59, 509000000, time.UTC),
		},
		{"foo", "2022-05-17T06:20:59.509Z", http.StatusBadRequest, 0, time.Time{}},
		{"70", "2022-05-17 06:20:59", http.StatusBadRequest, 0, time.Time{}},
	}

	for _, tc := range tc {
		mockLp := &mockLoadpoint{}

		handler := http.HandlerFunc(targetChargeHandler(mockLp))

		req, err := http.NewRequest("GET", fmt.Sprintf("/targetcharge/%s/%s", tc.inSoC, tc.inTime), nil)
		if err != nil {
			t.Fatal(err)
		}

		vars := map[string]string{
			"soc":  tc.inSoC,
			"time": tc.inTime,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != tc.statusCode {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, tc.statusCode)
		}

		if mockLp.SoC != tc.outSoC {
			t.Errorf("wrong target soc: got %v want %v", mockLp.SoC, tc.outSoC)
		}

		isoFormat := "2006-01-02T15:04:05.999Z07:00"
		if !mockLp.TargetTime.Equal(tc.outTime) {
			t.Errorf("wrong target time year: got %v want %v", mockLp.TargetTime.UTC().Format(isoFormat), tc.outTime.Format(isoFormat))
		}
	}
}
