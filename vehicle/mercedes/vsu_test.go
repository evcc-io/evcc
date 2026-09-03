package mercedes

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	pb "github.com/evcc-io/evcc/vehicle/mercedes/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMapVSUOdometer(t *testing.T) {
	vsu := &pb.VehicleStatusUpdate{
		FinOrVin: "WDB1234567890",
		Odo: &pb.Int64DistanceAttribute{
			Value: 42195,
			Unit:  pb.VehicleAttributeStatus_KILOMETERS,
		},
		Soc: &pb.Int64RatioAttribute{
			Value: 73,
		},
		Rangeelectric: &pb.Int64DistanceAttribute{
			Value: 210,
			Unit:  pb.VehicleAttributeStatus_KILOMETERS,
		},
		Chargingstatus: &pb.ChargingstatusEnumAttribute{
			Value: pb.Chargingstatus_CHARGINGSTATUS_AC_CHARGING_ACTIVE,
		},
	}

	res := mapVSU(vsu)

	assert.Equal(t, 42195, res.VehicleInfo.Odometer.Value, "odometer value")
	assert.Equal(t, "KILOMETERS", res.VehicleInfo.Odometer.Unit, "odometer unit")
	assert.Equal(t, 73.0, res.EvInfo.Battery.StateOfCharge, "soc")
	assert.Equal(t, 210, res.EvInfo.Battery.DistanceToEmpty.Value, "range")
	assert.Equal(t, pb.Chargingstatus_CHARGINGSTATUS_AC_CHARGING_ACTIVE, res.EvInfo.Battery.ChargingStatus)

	// AC charging active must map to StatusC (charging).
	assert.Equal(t, api.StatusC, MapChargeStatus(res.EvInfo.Battery.ChargingStatus))
}

// TestMapVSUOdometerWireRoundtrip proves the odo field (147) survives a
// marshal/unmarshal cycle – i.e. the regenerated proto decodes what the backend
// serialises.
func TestMapVSUOdometerWireRoundtrip(t *testing.T) {
	orig := &pb.VehicleStatusUpdate{
		Odo: &pb.Int64DistanceAttribute{
			Value: 123456,
			Unit:  pb.VehicleAttributeStatus_MILES,
		},
	}

	data, err := proto.Marshal(orig)
	require.NoError(t, err)

	var decoded pb.VehicleStatusUpdate
	require.NoError(t, proto.Unmarshal(data, &decoded))

	res := mapVSU(&decoded)
	assert.Equal(t, 123456, res.VehicleInfo.Odometer.Value)
	assert.Equal(t, "MILES", res.VehicleInfo.Odometer.Unit)
}

func TestMapVSUChargeCableUnplugged(t *testing.T) {
	// Missing chargingstatus falls back to UNPLUGGED (disconnected).
	res := mapVSU(&pb.VehicleStatusUpdate{})
	assert.Equal(t, pb.Chargingstatus_CHARGINGSTATUS_CHARGE_CABLE_UNPLUGGED, res.EvInfo.Battery.ChargingStatus)
	assert.Equal(t, api.StatusA, MapChargeStatus(res.EvInfo.Battery.ChargingStatus))
}

func TestMapVSUSocLimitFromChargeProgram(t *testing.T) {
	// No explicit max_soc: the SoC limit comes from the selected charge program.
	vsu := &pb.VehicleStatusUpdate{
		SelectedChargeProgram: &pb.SelectedChargeProgramEnumAttribute{
			Value: pb.SelectedChargeProgram_SELECTED_CHARGE_PROGRAM_HOME, // index 2
		},
		ChargePrograms: &pb.ChargeProgramsArrayAttribute{
			Value: []*pb.ChargeProgramParameters{
				{MaxSoc: 80},
				{MaxSoc: 90},
				{MaxSoc: 70}, // selected index 2
			},
		},
	}

	res := mapVSU(vsu)
	assert.Equal(t, 2, res.EvInfo.Battery.SelectedChargeProgram)
	assert.Equal(t, 70, res.EvInfo.Battery.SocLimit)
}

func TestMapVSUSocLimitExplicit(t *testing.T) {
	// Explicit max_soc wins over the charge-program fallback.
	vsu := &pb.VehicleStatusUpdate{
		MaxSoc: &pb.Int64RatioAttribute{Value: 85},
		ChargePrograms: &pb.ChargeProgramsArrayAttribute{
			Value: []*pb.ChargeProgramParameters{{MaxSoc: 70}},
		},
	}

	res := mapVSU(vsu)
	assert.Equal(t, 85, res.EvInfo.Battery.SocLimit)
}

// TestMapVSUSocLimitPresentButZero mirrors the live case: the VSU always carries
// max_soc, and a value of 0 means "no global override". A present-but-zero
// max_soc must fall through to the selected charge program instead of reporting
// a 0 % limit. (The legacy REST widget omitted the attribute entirely when
// unset, so this only surfaces over the VSU.)
func TestMapVSUSocLimitPresentButZero(t *testing.T) {
	vsu := &pb.VehicleStatusUpdate{
		MaxSoc: &pb.Int64RatioAttribute{Value: 0}, // present, unset
		SelectedChargeProgram: &pb.SelectedChargeProgramEnumAttribute{
			Value: pb.SelectedChargeProgram_SELECTED_CHARGE_PROGRAM_HOME, // index 2
		},
		ChargePrograms: &pb.ChargeProgramsArrayAttribute{
			Value: []*pb.ChargeProgramParameters{
				{MaxSoc: 100},
				{MaxSoc: 100},
				{MaxSoc: 100}, // selected index 2
			},
		},
	}

	res := mapVSU(vsu)
	assert.Equal(t, 100, res.EvInfo.Battery.SocLimit, "must fall back to charge program, not report 0%")
}

// TestMergeVSUReplacesFields verifies that a partial update replaces individual
// fields on the cached state while leaving untouched fields intact.
func TestMergeVSUReplacesFields(t *testing.T) {
	cached := &pb.VehicleStatusUpdate{
		Odo: &pb.Int64DistanceAttribute{Value: 1000, Unit: pb.VehicleAttributeStatus_KILOMETERS},
		Soc: &pb.Int64RatioAttribute{Value: 50},
	}

	// Partial update carrying only a new SoC.
	partial := &pb.VehicleStatusUpdate{
		Soc: &pb.Int64RatioAttribute{Value: 55},
	}

	mergeVSU(cached, partial)

	// Odo unchanged, Soc replaced.
	assert.Equal(t, int64(1000), cached.GetOdo().GetValue(), "odometer preserved")
	assert.Equal(t, int64(55), cached.GetSoc().GetValue(), "soc replaced")
}

// TestMergeVSUReplacesRepeated verifies repeated/array attributes are REPLACED,
// not concatenated (the reason we use protoreflect.Range instead of proto.Merge).
func TestMergeVSUReplacesRepeated(t *testing.T) {
	cached := &pb.VehicleStatusUpdate{
		ChargePrograms: &pb.ChargeProgramsArrayAttribute{
			Value: []*pb.ChargeProgramParameters{
				{MaxSoc: 80},
				{MaxSoc: 90},
			},
		},
	}

	partial := &pb.VehicleStatusUpdate{
		ChargePrograms: &pb.ChargeProgramsArrayAttribute{
			Value: []*pb.ChargeProgramParameters{
				{MaxSoc: 100},
			},
		},
	}

	mergeVSU(cached, partial)

	// The array must be replaced (1 entry), not appended (would be 3).
	require.Len(t, cached.GetChargePrograms().GetValue(), 1)
	assert.Equal(t, int32(100), cached.GetChargePrograms().GetValue()[0].GetMaxSoc())
}
