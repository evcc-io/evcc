package id

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed samples/status.json
var data []byte

func TestStatus(t *testing.T) {
	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		t.Error(err)
	}

	if v := status.Data.ChargingStatus.RemainingChargingTimeToCompleteMin; v != 120 {
		t.Error("invalid RemainingChargingTimeToCompleteMin", v)
	}
}
