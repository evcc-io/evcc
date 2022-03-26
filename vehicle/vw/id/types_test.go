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

	if status.Data.MaintenanceStatus != nil {
		t.Error("unexpected MaintenanceStatus")
	}
}

//go:embed samples/status2.json
var data2 []byte

func TestStatus2(t *testing.T) {
	var status Status
	if err := json.Unmarshal(data2, &status); err != nil {
		t.Error(err)
	}

	if status.Data.MaintenanceStatus == nil || status.Data.MaintenanceStatus.MileageKm == 0 {
		t.Error("invalid MaintenanceStatus.MileageKm")
	}
}
