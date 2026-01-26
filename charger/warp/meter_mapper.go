package warp

import "github.com/evcc-io/evcc/util"

type MeterMapper struct {
	indices MeterValuesIndices
	log     *util.Logger
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

	// check that all IDs are present
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

	// Create mapping
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
}

func (m *MeterMapper) UpdateValues(vals []float64, power *float64, energy *float64, voltL, currL *[3]float64) {
	// get the highest possible index
	highestIndex := max(m.indices.VoltageL1NIndex, m.indices.VoltageL2NIndex, m.indices.VoltageL3NIndex,
		m.indices.CurrentImExSumL1Index, m.indices.CurrentImExSumL2Index, m.indices.CurrentImExSumL3Index,
		m.indices.PowerImExSumIndex, m.indices.EnergyAbsImSumIndex)

	// check boundaries
	if len(vals) < highestIndex+1 {
		return
	}

	*voltL = [3]float64{vals[m.indices.VoltageL1NIndex], vals[m.indices.VoltageL2NIndex], vals[m.indices.VoltageL3NIndex]}
	*currL = [3]float64{vals[m.indices.CurrentImExSumL1Index], vals[m.indices.CurrentImExSumL2Index], vals[m.indices.CurrentImExSumL3Index]}
	*power = vals[m.indices.PowerImExSumIndex]
	*energy = vals[m.indices.EnergyAbsImSumIndex]
}
