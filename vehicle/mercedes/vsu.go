package mercedes

import (
	pb "github.com/evcc-io/evcc/vehicle/mercedes/pb"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// mergeVSU applies a partial VehicleStatusUpdate onto the cached full state.
//
// Mercedes sends a full_update once per (re)connect and partial updates
// afterwards. A partial update carries only the attributes that changed, so we
// must replace those fields on the cached state field-by-field. We deliberately
// do NOT use proto.Merge: for repeated/array attributes (e.g. charge_programs)
// proto.Merge appends instead of replacing, which would corrupt the state.
//
// protoreflect.Range only visits fields that are set on src, so any attribute
// present in the partial update replaces the corresponding field on dst while
// everything else is left untouched.
func mergeVSU(dst, src *pb.VehicleStatusUpdate) {
	if dst == nil || src == nil {
		return
	}

	dstReflect := dst.ProtoReflect()
	src.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dstReflect.Set(fd, v)
		return true
	})
}

// mapVSU maps a decoded VehicleStatusUpdate into evcc's StatusResponse shape.
//
// It reads at least the odometer (odo, field 147), state of charge, electric
// range, charging status, preconditioning, position and the effective SoC
// limit. Only attributes actually present in the update are populated; the
// caller decides freshness/validity.
func mapVSU(vsu *pb.VehicleStatusUpdate) StatusResponse {
	var res StatusResponse
	if vsu == nil {
		return res
	}

	// Odometer (Int64DistanceAttribute, field 147) – the primary reason for the
	// VSU migration. Unit string matches the legacy value (KILOMETERS/MILES).
	if odo := vsu.GetOdo(); odo != nil {
		res.VehicleInfo.Odometer.Value = int(odo.GetValue())
		res.VehicleInfo.Odometer.Unit = odo.GetUnit().String()
		if ts := odo.GetMetadata().GetTimestamp(); ts != nil {
			res.VehicleInfo.Timestamp = ts.AsTime()
		}
	}

	// State of charge (Int64RatioAttribute, field 196), percent.
	if soc := vsu.GetSoc(); soc != nil {
		res.EvInfo.Battery.StateOfCharge = float64(soc.GetValue())
	}

	// Electric range (Int64DistanceAttribute, field 183).
	if r := vsu.GetRangeelectric(); r != nil {
		res.EvInfo.Battery.DistanceToEmpty.Value = int(r.GetValue())
		res.EvInfo.Battery.DistanceToEmpty.Unit = r.GetUnit().String()
	}

	// End of charge time (Int64ClockHourAttribute, field 96), minutes after
	// midnight.
	if e := vsu.GetEndofchargetime(); e != nil {
		res.EvInfo.Battery.EndOfChargeTime = int(e.GetValue())
	}

	// Charging status (EnumAttribute, field 50). The proto enum values match
	// the legacy integer codes consumed by MapChargeStatus. When the attribute
	// is absent we fall back to UNPLUGGED (3, disconnected), matching the
	// previous REST behaviour.
	if cs := vsu.GetChargingstatus(); cs != nil {
		res.EvInfo.Battery.ChargingStatus = cs.GetValue()
	} else {
		res.EvInfo.Battery.ChargingStatus = pb.Chargingstatus_CHARGINGSTATUS_CHARGE_CABLE_UNPLUGGED
	}

	// SoC limit: prefer the explicit max_soc (Int64RatioAttribute, field 138);
	// otherwise fall back to the max_soc of the currently selected charge
	// program (charge_programs array, field 27, indexed by selected_charge_program,
	// field 190).
	//
	// Unlike the legacy REST widget – where an unset limit meant the maxSoc
	// attribute was simply absent – the VSU always carries max_soc, and a value
	// of 0 means "no global override, use the per-program limit". So a
	// present-but-zero max_soc must fall through to the charge program rather
	// than being reported as a 0 % limit (observed live: max_soc=0 while every
	// charge program reports 100).
	if sel := vsu.GetSelectedChargeProgram(); sel != nil {
		res.EvInfo.Battery.SelectedChargeProgram = int(sel.GetValue())
	}
	if m := vsu.GetMaxSoc(); m != nil && m.GetValue() > 0 {
		res.EvInfo.Battery.SocLimit = int(m.GetValue())
	} else if cps := vsu.GetChargePrograms().GetValue(); len(cps) > 0 {
		idx := res.EvInfo.Battery.SelectedChargeProgram
		if idx >= 0 && idx < len(cps) && cps[idx] != nil {
			res.EvInfo.Battery.SocLimit = int(cps[idx].GetMaxSoc())
		}
	}

	// Preconditioning: precond_active (BoolAttribute, field 167). precond_now
	// became an enum in the VSU model, so we rely on precond_active only.
	if p := vsu.GetPrecondActive(); p != nil {
		res.Preconditioning.Active = p.GetValue()
	}

	// Position (DoubleAttribute, fields 165/166).
	if lat := vsu.GetPositionLat(); lat != nil {
		res.LocationResponse.Latitude = lat.GetValue()
	}
	if lng := vsu.GetPositionLong(); lng != nil {
		res.LocationResponse.Longitude = lng.GetValue()
	}

	return res
}
