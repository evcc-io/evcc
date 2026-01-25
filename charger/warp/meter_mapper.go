package warp

import "github.com/evcc-io/evcc/util"

type MeterMapper struct {
	indices    MeterValuesIndices
	skipLegacy bool
	log        *util.Logger
}

func (m *MeterMapper) HandleLegacyValues(vals []float64, power *float64, energy *float64, voltL, currL *[3]float64) {
	if m.skipLegacy {
		return
	}
	if len(vals) >= 6 {
		(*voltL)[0], (*voltL)[1], (*voltL)[2] = vals[0], vals[1], vals[2]
		(*currL)[0], (*currL)[1], (*currL)[2] = vals[3], vals[4], vals[5]
	}
}

func (m *MeterMapper) UpdateValueIDs(ids []int) {
	required := []int{
		ValueIDVoltageL1N,
		ValueIDVoltageL2N,
		ValueIDVoltageL3N,
		ValueIDCurrentImExSumL1,
		ValueIDCurrentImExSumL2,
		ValueIDCurrentImExSumL3,
		ValueIDPowerImExSum,
		ValueIDEnergyAbsImSum,
	}

	// prüfen ob alle IDs vorhanden sind
	missing := []int{}
	for _, req := range required {
		found := false
		for _, id := range ids {
			if id == req {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, req)
		}
	}

	if len(missing) > 0 {
		m.log.ERROR.Printf("missing required meter value IDs: %v", missing)
		return
	}

	// Mapping erzeugen
	var idx MeterValuesIndices
	for i, valueID := range ids {
		switch valueID {
		case ValueIDVoltageL1N:
			idx.VoltageL1NIndex = i
		case ValueIDVoltageL2N:
			idx.VoltageL2NIndex = i
		case ValueIDVoltageL3N:
			idx.VoltageL3NIndex = i
		case ValueIDCurrentImExSumL1:
			idx.CurrentImExSumL1Index = i
		case ValueIDCurrentImExSumL2:
			idx.CurrentImExSumL2Index = i
		case ValueIDCurrentImExSumL3:
			idx.CurrentImExSumL3Index = i
		case ValueIDPowerImExSum:
			idx.PowerImExSumIndex = i
		case ValueIDEnergyAbsImSum:
			idx.EnergyAbsImSumIndex = i
		}
	}

	m.indices = idx
	//m.log.INFO.Printf("meter value_ids mapped: %+v", idx)
}

func (m *MeterMapper) UpdateValues(vals []float64, power *float64, energy *float64, voltL, currL *[3]float64) {
	highestIndex := max(m.indices.VoltageL1NIndex, m.indices.VoltageL2NIndex, m.indices.VoltageL3NIndex,
		m.indices.CurrentImExSumL1Index, m.indices.CurrentImExSumL2Index, m.indices.CurrentImExSumL3Index,
		m.indices.PowerImExSumIndex, m.indices.EnergyAbsImSumIndex)

	if len(vals) < highestIndex+1 {
		return
	}

	voltL[0] = vals[m.indices.VoltageL1NIndex]
	voltL[1] = vals[m.indices.VoltageL2NIndex]
	voltL[2] = vals[m.indices.VoltageL3NIndex]
	currL[0] = vals[m.indices.CurrentImExSumL1Index]
	currL[1] = vals[m.indices.CurrentImExSumL2Index]
	currL[2] = vals[m.indices.CurrentImExSumL3Index]
	power = &vals[m.indices.PowerImExSumIndex]
	energy = &vals[m.indices.EnergyAbsImSumIndex]
}
