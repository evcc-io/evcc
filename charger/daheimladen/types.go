package daheimladen

type GetLatestStatus struct {
	ChargingStationID    string  `json:"charging_station_id"`
	Status               string  `json:"status"`
	ActivePowerImport    float32 `json:"active_power_import"`
	TotalMeterValue      int64   `json:"total_meter_value"`
	VoltagePhaseL1L2     float32 `json:"voltage_phase_l1l2"`
	VoltagePhaseL2L3     float32 `json:"voltage_phase_l2l3"`
	VoltagePhaseL3L1     float32 `json:"voltage_phase_l3l1"`
	CurrentImportPhaseL1 float32 `json:"current_import_phase_l1"`
	CurrentImportPhaseL2 float32 `json:"current_import_phase_l2"`
	CurrentImportPhaseL3 float32 `json:"current_import_phase_l3"`
	Temperature          int32   `json:"charging_station_temperature"`
	CurrentTime          string  `json:"current_time"`
	TransactionStartAt   string  `json:"transaction_start_at"`
	TranasctionPowerUsed int64   `json:"transaction_power_used"`
}

type GetLatestInProgressTransactionResponse struct {
	TransactionID int32 `json:"transaction_id"`
}

type GetLatestMeterValueResponse struct {
	ChargingStationID          string  `json:"charging_station_id"`
	ConnectorID                int32   `json:"connector_id"`
	PowerActiveImport          float32 `json:"power_active_import"`
	CurrentImportPhaseL1       float32 `json:"current_import_phasel1"`
	CurrentImportPhaseL2       float32 `json:"current_import_phasel2"`
	CurrentImportPhaseL3       float32 `json:"current_import_phasel3"`
	EnergyActiveImportRegister float32 `json:"energy_active_import_register"`
	Timestamp                  string  `json:"timestamp"`
}

type RemoteStartRequest struct {
	ConnectorID int32  `json:"connector_id"`
	IdTag       string `json:"idtag"`
}

type RemoteStartResponse struct {
	Status string `json:"status"`
}

type RemoteStopRequest struct {
	TransactionID int32 `json:"transaction_id"`
}

type RemoteStopResponse struct {
	Status string `json:"status"`
}

type ChangeConfigurationRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ChangeConfigurationResponse struct {
	Status string `json:"status"`
}

type GetConfigurationRequest struct {
	Key string `json:"key"`
}

type GetConfigurationResponse struct {
	ReadOnly bool   `json:"readonly"`
	Value    string `json:"value"`
}
