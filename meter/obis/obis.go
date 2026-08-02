package obis

// OBIS codes for electricity meters.
// https://www.kbr.de/de/obis-kennzeichen/elektrizitaet
//
// Names use Import (+A, OBIS channel 1/2x) and Export (-A, channel 2/2x) for
// active power and energy. Energy registers carry an optional tariff suffix
// T1/T2 (OBIS field E).
//
// The OBIS D field selects how the value is processed:
//   - D=7  instantaneous    momentary sample ("now")               -> unqualified
//   - D=4  current average  running mean over the ongoing demand   -> Demand suffix
//                           period (a meter-configured window, e.g. 15 min)
//   - D=5  last average     mean over the last completed period
//   - D=8  time integral    cumulative energy                      -> Energy*

// Active power (kW)
const (
	PowerImport = "1-0:1.7.0" // instantaneous +A
	PowerExport = "1-0:2.7.0" // instantaneous -A

	PowerImportL1 = "1-0:21.7.0"
	PowerImportL2 = "1-0:41.7.0"
	PowerImportL3 = "1-0:61.7.0"

	PowerExportL1 = "1-0:22.7.0"
	PowerExportL2 = "1-0:42.7.0"
	PowerExportL3 = "1-0:62.7.0"

	PowerImportDemand = "1-0:1.4.0" // current average +A
	PowerExportDemand = "1-0:2.4.0" // current average -A

	PowerImportDemandL1 = "1-0:21.4.0"
	PowerImportDemandL2 = "1-0:41.4.0"
	PowerImportDemandL3 = "1-0:61.4.0"
)

// Active energy (kWh)
const (
	EnergyImport   = "1-0:1.8.0" // +A, total
	EnergyImportT1 = "1-0:1.8.1" // +A, tariff 1
	EnergyImportT2 = "1-0:1.8.2" // +A, tariff 2

	EnergyExport   = "1-0:2.8.0" // -A, total
	EnergyExportT1 = "1-0:2.8.1" // -A, tariff 1
	EnergyExportT2 = "1-0:2.8.2" // -A, tariff 2
)

// Current (A)
const (
	CurrentL1 = "1-0:31.7.0" // instantaneous
	CurrentL2 = "1-0:51.7.0"
	CurrentL3 = "1-0:71.7.0"

	CurrentDemandL1 = "1-0:31.4.0" // current average
	CurrentDemandL2 = "1-0:51.4.0"
	CurrentDemandL3 = "1-0:71.4.0"
)

// Voltage (V)
const (
	VoltageL1 = "1-0:32.7.0" // instantaneous
	VoltageL2 = "1-0:52.7.0"
	VoltageL3 = "1-0:72.7.0"

	VoltageDemandL1 = "1-0:32.4.0" // current average
	VoltageDemandL2 = "1-0:52.4.0"
	VoltageDemandL3 = "1-0:72.4.0"
)
