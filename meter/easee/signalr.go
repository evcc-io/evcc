package easee

import (
	"strconv"
	"time"
)

type Observation struct {
	Mid       string
	DataType  DataType
	ID        ObservationID
	Timestamp time.Time
	Value     string
}

func (o *Observation) TypedValue() (any, error) {
	switch o.DataType {
	case Boolean:
		return o.Value == "1", nil
	case Double:
		return strconv.ParseFloat(o.Value, 64)
	case Integer:
		return strconv.Atoi(o.Value)
	case String:
		fallthrough
	default:
		return o.Value, nil
	}
}

type SignalRCommandResponse struct {
	SerialNumber string
	ID           int
	Timestamp    time.Time
	DeliveredAt  time.Time
	WasAccepted  bool
	ResultCode   int
	Comment      string
	Ticks        int64
}

type RestCommandResponse struct {
	Device    string
	CommandId int
	Ticks     int64
}

type DataType int

// https://github.com/Masterloop/Masterloop.Core.Types/blob/master/src/Masterloop.Core.Types/Base/DataType.cs
const (
	_          DataType = iota
	Binary              // 1
	Boolean             // 2
	Double              // 3
	Integer             // 4
	Position            // 5
	String              // 6
	Statistics          // 7
)

// https://developer.easee.com/docs/equalizer-observations
type ObservationID int

//go:generate go tool enumer -type ObservationID
const (
	SELF_TEST_RESULT                 ObservationID = 1  // PASSED or error codes [String]
	SELF_TEST_DETAILS                ObservationID = 2  // JSON with details from self-test [String]
	EASEE_LINK_COMMAND_RESPONSE      ObservationID = 13 // Response on an EaseeLink command [Integer]
	EASEE_LINK_DATA_RECEIVED         ObservationID = 14 // Data received on EaseeLink [String]
	SITE_ID_NUMERIC                  ObservationID = 19 // Site ID numeric value [event]
	SITE_STRUCTURE                   ObservationID = 20 // Site structure [boot]
	SOFTWARE_RELEASE                 ObservationID = 21 // Software release [boot]
	DEVICE_MODE                      ObservationID = 23 // Current device mode
	METER_TYPE                       ObservationID = 25 // Meter type
	METER_ID                         ObservationID = 26 // Meter identification
	OBIS_LIST_IDENTIFIER             ObservationID = 27 // OBIS list identifier
	GRID_TYPE                        ObservationID = 29 // Grid type
	NUM_PHASES                       ObservationID = 30 // Number of phases
	CURRENT_L1                       ObservationID = 31 // Current L1 [A]
	CURRENT_L2                       ObservationID = 32 // Current L2 [A]
	CURRENT_L3                       ObservationID = 33 // Current L3 [A]
	VOLTAGE_N_L1                     ObservationID = 34 // Voltage N-L1 [V]
	VOLTAGE_N_L2                     ObservationID = 35 // Voltage N-L2 [V]
	VOLTAGE_N_L3                     ObservationID = 36 // Voltage N-L3 [V]
	VOLTAGE_L1_L2                    ObservationID = 37 // Voltage L1-L2 [V]
	VOLTAGE_L1_L3                    ObservationID = 38 // Voltage L1-L3 [V]
	VOLTAGE_L2_L3                    ObservationID = 39 // Voltage L2-L3 [V]
	ACTIVE_POWER_IMPORT              ObservationID = 40 // Active power import [kW]
	ACTIVE_POWER_EXPORT              ObservationID = 41 // Active power export [kW]
	REACTIVE_POWER_IMPORT            ObservationID = 42 // Reactive power import [kVAR]
	REACTIVE_POWER_EXPORT            ObservationID = 43 // Reactive power export [kVAR]
	MAX_POWER_IMPORT                 ObservationID = 44 // Max power import [event]
	CUMULATIVE_ACTIVE_POWER_IMPORT   ObservationID = 45 // Cumulative active power import [kWh]
	CUMULATIVE_ACTIVE_POWER_EXPORT   ObservationID = 46 // Cumulative active power export [kWh]
	CUMULATIVE_REACTIVE_POWER_IMPORT ObservationID = 47 // Cumulative reactive power import [kVARh]
	CUMULATIVE_REACTIVE_POWER_EXPORT ObservationID = 48 // Cumulative reactive power export [kVARh]
	CLOCK_AND_DATE_METER             ObservationID = 49 // Meter clock/date
	RSSI                             ObservationID = 50 // RSSI
	SSID                             ObservationID = 51 // WiFi SSID
	MASTER_BACK_PLATE_ID             ObservationID = 55 // Master back plate ID [event]
	EQUALIZER_ID                     ObservationID = 56 // Equalizer back plate RFID [boot]
	SITE_AND_CIRCUIT_STRUCTURE       ObservationID = 57 // Site/circuit structure payload

	// Additional IDs seen on stream in the field.
	NETWORK_SIGNAL_RSSI           ObservationID = 70
	METER_COMMUNICATION_CONFIG    ObservationID = 85
	DEFAULT_LEVEL                 ObservationID = 86
	AVAILABLE_CURRENT_P1          ObservationID = 87
	AVAILABLE_CURRENT_P2          ObservationID = 88
	AVAILABLE_CURRENT_P3          ObservationID = 89
	RADIO_ACCESS_TECHNOLOGY       ObservationID = 100
	PHASE_RELAY_STATUS_P1         ObservationID = 105
	PHASE_RELAY_STATUS_P2         ObservationID = 106
	PHASE_RELAY_STATUS_P3         ObservationID = 107
	SECURITY_OR_ENCRYPTION_STATUS ObservationID = 111
	OPERATING_MODE                ObservationID = 115
	AMP_CLAMP_STATUS              ObservationID = 120
	CONNECTED_TO_CLOUD            ObservationID = 250
	CLOUD_DISCONNECT_REASON       ObservationID = 251
)
