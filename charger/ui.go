package charger

// SupportedReadings is the set of supported readings for the integrated meter
type SupportedReadings struct {
	Power    bool `ui:"de=Leistung (W)"`
	Energy   bool `ui:"de=Zählerstand (kWh)"`
	Currents bool `ui:"de=Strom (A)"`
}
