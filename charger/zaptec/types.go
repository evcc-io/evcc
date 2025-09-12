package zaptec

import (
	"strconv"
)

type ChargersResponse struct {
	Pages int
	Data  []Charger
}

type Charger struct {
	OperatingMode           int
	IsOnline                bool
	Id                      string
	MID                     string
	DeviceId                string
	SerialNo                string
	Name                    string
	CreatedOnDate           string
	CircuitId               string
	Active                  bool
	CurrentUserRoles        int
	DeviceType              int
	InstallationName        string
	InstallationId          string
	AuthenticationType      int
	IsAuthorizationRequired bool
}

type StateResponse []Observation

func (s *StateResponse) ObservationByID(id ObservationID) *Observation {
	if s == nil {
		return nil
	}

	for _, o := range *s {
		if o.StateId == id {
			return &o
		}
	}

	return nil
}

type Observation struct {
	ChargerId     string
	StateId       ObservationID
	Timestamp     string
	ValueAsString string
}

func (o *Observation) Bool() bool {
	return o != nil && (o.ValueAsString == "true" || o.ValueAsString == "1")
}

func (o *Observation) Int() (int, error) {
	if o == nil || o.ValueAsString == "" {
		return 0, nil
	}

	return strconv.Atoi(o.ValueAsString)
}

func (o *Observation) Float64() (float64, error) {
	if o == nil || o.ValueAsString == "" {
		return 0, nil
	}

	return strconv.ParseFloat(o.ValueAsString, 64)
}

type Update struct {
	MaxChargeCurrent     *int `json:"maxChargeCurrent,omitempty"`
	MaxChargePhases      *int `json:"maxChargePhases,omitempty"`
	MinChargeCurrent     *int `json:"minChargeCurrent,omitempty"`
	OfflineChargeCurrent *int `json:"offlineChargeCurrent,omitempty"`
	OfflineChargePhase   *int `json:"offlineChargePhase,omitempty"`
	MeterValueInterval   *int `json:"meterValueInterval,omitempty"`
}

type SessionPriority struct {
	PrioritizedPhases *int `json:"prioritizedPhases,omitempty"`
}

type Installation struct {
	Id         string  `json:"id"`
	MaxCurrent float64 `json:"maxCurrent"`
}

type UpdateInstallation struct {
	AvailableCurrentPhase1 *int `json:"availableCurrentPhase1,omitempty"`
	AvailableCurrentPhase2 *int `json:"availableCurrentPhase2,omitempty"`
	AvailableCurrentPhase3 *int `json:"availableCurrentPhase3,omitempty"`
}

type CapabilitiesResponse struct {
	CommunicationModes []string `json:"communicationModes"`
	DeviceType         string   `json:"deviceType"`
	MeterCalibrated    bool     `json:"meterCalibrated"`
	PhaseBalancing     []string `json:"phaseBalancing"`
	ProductVariant     string   `json:"productVariant"`
	SchemaVersion      string   `json:"schemaVersion"`
}
