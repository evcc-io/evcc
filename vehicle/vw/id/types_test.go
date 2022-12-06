package id

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed samples/selectivestatus.json
var data []byte

func TestStatus(t *testing.T) {
	var status SelectiveSatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Error(err)
	}

	if v := status.Charging.ChargingStatus.Value.RemainingChargingTimeToCompleteMin; v != 5 {
		t.Error("invalid RemainingChargingTimeToCompleteMin", v)
	}

}
