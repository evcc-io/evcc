package server

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type mockLoadpoint struct {
	Soc        int
	TargetTime time.Time
}

func (lp *mockLoadpoint) SetTargetCharge(time time.Time, soc int) error {
	lp.Soc = soc
	lp.TargetTime = time
	return nil
}

func TestTargetChargeHandler(t *testing.T) {
	tc := []struct {
		inSoc      string
		inTime     string
		statusCode int
		outSoc     int
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

		req, err := http.NewRequest("GET", fmt.Sprintf("/targetcharge/%s/%s", tc.inSoc, tc.inTime), nil)
		if err != nil {
			t.Fatal(err)
		}

		vars := map[string]string{
			"soc":  tc.inSoc,
			"time": tc.inTime,
		}

		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != tc.statusCode {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, tc.statusCode)
		}

		if mockLp.Soc != tc.outSoc {
			t.Errorf("wrong target soc: got %v want %v", mockLp.Soc, tc.outSoc)
		}

		isoFormat := "2006-01-02T15:04:05.999Z07:00"
		if !mockLp.TargetTime.Equal(tc.outTime) {
			t.Errorf("wrong target time year: got %v want %v", mockLp.TargetTime.UTC().Format(isoFormat), tc.outTime.Format(isoFormat))
		}
	}
}

func TestNaNInf(t *testing.T) {
	c := map[string]any{
		"foo": math.NaN(),
		"bar": math.Inf(0),
	}
	encodeFloats(c)
	assert.Equal(t, map[string]any{"foo": nil, "bar": nil}, c, "NaN not encoded as nil")
}
