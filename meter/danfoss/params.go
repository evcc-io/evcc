package danfoss

// TLX parameter IDs. Sourced from AMajland/Danfoss-TLX (confirmed on TLX 6/8/
// 10/12.5/15 kW hardware) and cross-checked against yvesf/pycomlynx.
const (
	ParamTotalEnergy uint16 = 0x0102 // lifetime production, raw value in Wh

	ParamGridPowerTotal uint16 = 0x0246 // W
	ParamGridPowerL1    uint16 = 0x0242 // W
	ParamGridPowerL2    uint16 = 0x0243 // W
	ParamGridPowerL3    uint16 = 0x0244 // W

	ParamGridVoltageL1 uint16 = 0x023c // V * 10
	ParamGridVoltageL2 uint16 = 0x023d // V * 10
	ParamGridVoltageL3 uint16 = 0x023e // V * 10

	ParamGridCurrentL1 uint16 = 0x023f // A * 1000 (mA)
	ParamGridCurrentL2 uint16 = 0x0240 // A * 1000 (mA)
	ParamGridCurrentL3 uint16 = 0x0241 // A * 1000 (mA)

	ParamOperatingMode uint16 = 0x0a02 // enum (0-9=off, 10-49=boot, 50-59=connecting, 60-69=on-grid, 70-79=failsafe, 80-89=off-comm)

	// Internal aliases used within this package. Keep in sync with the
	// exported names above so the golden-frame test can reference them without
	// an import cycle.
	paramTotalEnergy    = ParamTotalEnergy
	paramGridPowerTotal = ParamGridPowerTotal
	paramGridPowerL1    = ParamGridPowerL1
	paramGridVoltageL1  = ParamGridVoltageL1
	paramGridCurrentL1  = ParamGridCurrentL1
	paramOperatingMode  = ParamOperatingMode
)
