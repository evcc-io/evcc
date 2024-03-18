package hello

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed VehicleStatus.json
var data []byte

func TestVehicleStatus(t *testing.T) {
	var res VehicleStatus
	require.NoError(t, json.Unmarshal(data, &res))
}
