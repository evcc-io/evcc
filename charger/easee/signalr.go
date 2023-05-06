package easee

import "time"

type Observation struct {
	Mid       string
	DataType  DataType
	ID        ObservationID
	Timestamp time.Time
	Value     string
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

// https://www.notion.so/Charger-template-c6a20ff7cfea41e2b5f80b00afb34af5
type ObservationID int

//go:generate enumer -type ObservationID
const (
	SELF_TEST_RESULT                                   ObservationID = 1   // PASSED or error codes [String]
	SELF_TEST_DETAILS                                  ObservationID = 2   // JSON with details from self-test [String]
	WIFI_EVENT                                         ObservationID = 10  // Enum with WiFi event codes. Requires telemetry debug mode. Will be updated on WiFi events when using cellular,  will otherwise be reported in ChargerOfflineReason [Integer]
	CHARGER_OFFLINE_REASON                             ObservationID = 11  // Enum describing why charger is offline [Integer]
	EASEE_LINK_COMMAND_RESPONSE                        ObservationID = 13  // Response on a EaseeLink command sent to another devic [Integer]
	EASEE_LINK_DATA_RECEIVED                           ObservationID = 14  // Data received on EaseeLink from another device [String]
	LOCAL_PRE_AUTHORIZE_ENABLED                        ObservationID = 15  // Preauthorize with whitelist enabled. Readback on setting [event] [Boolean]
	LOCAL_AUTHORIZE_OFFLINE_ENABLED                    ObservationID = 16  // Allow offline charging for whitelisted RFID token. Readback on setting [event] [Boolean]
	ALLOW_OFFLINE_TX_FOR_UNKNOWN_ID                    ObservationID = 17  // Allow offline charging for all RFID tokens. Readback on setting [event] [Boolean]
	ERRATIC_EVMAX_TOGGLES                              ObservationID = 18  // 0 == erratic checking disabled, otherwise the number of toggles between states Charging and Charging Complate that will trigger an error [Integer]
	BACKPLATE_TYPE                                     ObservationID = 19  // Readback on backplate type [Integer]
	SITE_STRUCTURE                                     ObservationID = 20  // Site Structure [boot] [String]
	DETECTED_POWER_GRID_TYPE                           ObservationID = 21  // Detected power grid type according to PowerGridType table [boot] [Integer]
	CIRCUIT_MAX_CURRENT_P1                             ObservationID = 22  // Set circuit maximum current [Amperes] [Double]
	CIRCUIT_MAX_CURRENT_P2                             ObservationID = 23  // Set circuit maximum current [Amperes] [Double]
	CIRCUIT_MAX_CURRENT_P3                             ObservationID = 24  // Set circuit maximum current [Amperes] [Double]
	LOCATION                                           ObservationID = 25  // Location coordinate [event] [Position]
	SITE_IDSTRING                                      ObservationID = 26  // Site ID string [event] [String]
	SITE_IDNUMERIC                                     ObservationID = 27  // Site ID numeric value [event] [Integer]
	LOCK_CABLE_PERMANENTLY                             ObservationID = 30  // Lock type2 cable permanently [Boolean]
	IS_ENABLED                                         ObservationID = 31  // Set true to enable charger, false disables charger [Boolean]
	CIRCUIT_SEQUENCE_NUMBER                            ObservationID = 33  // Charger sequence number on circuit [Integer]
	SINGLE_PHASE_NUMBER                                ObservationID = 34  // Phase to use in 1-phase charging [Integer]
	ENABLE3_PHASES_DEPRECATED                          ObservationID = 35  // Allow charging using 3-phases [Boolean]
	WI_FI_SSID                                         ObservationID = 36  // WiFi SSID name [String]
	ENABLE_IDLE_CURRENT                                ObservationID = 37  // Charger signals available current when EV is done charging [user option] [event] [Boolean]
	PHASE_MODE                                         ObservationID = 38  // Phase mode on this charger. 1-Locked to 1-Phase, 2-Auto, 3-Locked to 3-phase(only Home) [Integer]
	FORCED_THREE_PHASE_ON_ITWITH_GND_FAULT             ObservationID = 39  // Default disabled. Must be set manually if grid type is indeed three phase IT [Boolean]
	LED_STRIP_BRIGHTNESS                               ObservationID = 40  // LED strip brightness, 0-100% [Integer]
	LOCAL_AUTHORIZATION_REQUIRED                       ObservationID = 41  // Local RFID authorization is required for charging [user options] [event] [Boolean]
	AUTHORIZATION_REQUIRED                             ObservationID = 42  // Authorization is requried for charging [Boolean]
	REMOTE_START_REQUIRED                              ObservationID = 43  // Remote start required flag [event] [Boolean]
	SMART_BUTTON_ENABLED                               ObservationID = 44  // Smart button is enabled [Boolean]
	OFFLINE_CHARGING_MODE                              ObservationID = 45  // Charger behavior when offline [Integer]
	LEDMODE                                            ObservationID = 46  // Charger LED mode [event] [Integer]
	MAX_CHARGER_CURRENT                                ObservationID = 47  // Max current this charger is allowed to offer to car (A). Non volatile. [Double]
	DYNAMIC_CHARGER_CURRENT                            ObservationID = 48  // Max current this charger is allowed to offer to car (A). Volatile [Double]
	MAX_CURRENT_OFFLINE_FALLBACK_P1                    ObservationID = 50  // Maximum circuit current P1 when offline [event] [Integer]
	MAX_CURRENT_OFFLINE_FALLBACK_P2                    ObservationID = 51  // Maximum circuit current P2 when offline [event] [Integer]
	MAX_CURRENT_OFFLINE_FALLBACK_P3                    ObservationID = 52  // Maximum circuit current P3 when offline [event] [Integer]
	CHARGING_SCHEDULE                                  ObservationID = 62  // Charging schedule [json] [String]
	PAIRED_EQUALIZER                                   ObservationID = 65  // Paired equalizer details [String]
	WI_FI_APENABLED                                    ObservationID = 68  // True if WiFi Access Point is enabled, otherwise false [Boolean]
	PAIRED_USER_IDTOKEN                                ObservationID = 69  // Observed user token when charger put in RFID pairing mode [event] [String]
	CIRCUIT_TOTAL_ALLOCATED_PHASE_CONDUCTOR_CURRENT_L1 ObservationID = 70  // Total current allocated to L1 by all chargers on the circuit. Sent in by master only [Double]
	CIRCUIT_TOTAL_ALLOCATED_PHASE_CONDUCTOR_CURRENT_L2 ObservationID = 71  // Total current allocated to L2 by all chargers on the circuit. Sent in by master only [Double]
	CIRCUIT_TOTAL_ALLOCATED_PHASE_CONDUCTOR_CURRENT_L3 ObservationID = 72  // Total current allocated to L3 by all chargers on the circuit. Sent in by master only [Double]
	CIRCUIT_TOTAL_PHASE_CONDUCTOR_CURRENT_L1           ObservationID = 73  // Total current in L1 (sum of all chargers on the circuit) Sent in by master only [Double]
	CIRCUIT_TOTAL_PHASE_CONDUCTOR_CURRENT_L2           ObservationID = 74  // Total current in L2 (sum of all chargers on the circuit) Sent in by master only [Double]
	CIRCUIT_TOTAL_PHASE_CONDUCTOR_CURRENT_L3           ObservationID = 75  // Total current in L3 (sum of all chargers on the circuit) Sent in by master only [Double]
	NUMBER_OF_CARS_CONNECTED                           ObservationID = 76  // Number of cars connected to this circuit [Integer]
	NUMBER_OF_CARS_CHARGING                            ObservationID = 77  // Number of cars currently charging [Integer]
	NUMBER_OF_CARS_IN_QUEUE                            ObservationID = 78  // Number of cars currently in queue, waiting to be allocated power [Integer]
	NUMBER_OF_CARS_FULLY_CHARGED                       ObservationID = 79  // Number of cars that appear to be fully charged [Integer]
	SOFTWARE_RELEASE                                   ObservationID = 80  // Embedded software package release id [boot] [Integer]
	ICCID                                              ObservationID = 81  // SIM integrated circuit card identifier [String]
	MODEM_FW_ID                                        ObservationID = 82  // Modem firmware version [String]
	OTAERROR_CODE                                      ObservationID = 83  // OTA error code, see table [event] [Integer]
	MOBILE_NETWORK_OPERATOR                            ObservationID = 84  // Current mobile network operator [pollable] [String]
	REBOOT_REASON                                      ObservationID = 89  // Reason of reboot. Bitmask of flags. [Integer]
	POWER_PCBVERSION                                   ObservationID = 90  // Power PCB hardware version [Integer]
	COM_PCBVERSION                                     ObservationID = 91  // Communication PCB hardware version [Integer]
	REASON_FOR_NO_CURRENT                              ObservationID = 96  // Enum describing why a charger with a car connected is not offering current to the car [Integer]
	LOAD_BALANCING_NUMBER_OF_CONNECTED_CHARGERS        ObservationID = 97  // Number of connected chargers in the load balancin. Including the master. Sent from Master only. [Integer]
	UDPNUM_OF_CONNECTED_NODES                          ObservationID = 98  // Number of chargers connected to master through UDP / WIFI [Integer]
	LOCAL_CONNECTION                                   ObservationID = 99  // Slaves only. Current connection to master, 0 = None, 1= Radio, 2 = WIFI UDP, 3 = Radio and WIFI UDP [Integer]
	PILOT_MODE                                         ObservationID = 100 // Pilot Mode Letter (A-F) [event] [String]
	CAR_CONNECTED_DEPRECATED                           ObservationID = 101 // Car connection state [Boolean]
	SMART_CHARGING                                     ObservationID = 102 // Smart charging state enabled by capacitive touch button [event] [Boolean]
	CABLE_LOCKED                                       ObservationID = 103 // Cable lock state [event] [Boolean]
	CABLE_RATING                                       ObservationID = 104 // Cable rating read [Amperes] [event] [Double]
	PILOT_HIGH                                         ObservationID = 105 // Pilot signal high [Volt] [debug] [Double]
	PILOT_LOW                                          ObservationID = 106 // Pilot signal low [Volt] [debug] [Double]
	BACK_PLATE_ID                                      ObservationID = 107 // Back Plate RFID of charger [boot] [String]
	USER_IDTOKEN_REVERSED                              ObservationID = 108 // User ID token string from RFID reading [event] (NB! Must reverse these strings) [String]
	CHARGER_OP_MODE                                    ObservationID = 109 // Charger operation mode according to charger mode table [event] [Integer]
	OUTPUT_PHASE                                       ObservationID = 110 // Active output phase(s) to EV according to output phase type table. [event] [Integer]
	DYNAMIC_CIRCUIT_CURRENT_P1                         ObservationID = 111 // Dynamically set circuit maximum current for phase 1 [Amperes] [event] [Double]
	DYNAMIC_CIRCUIT_CURRENT_P2                         ObservationID = 112 // Dynamically set circuit maximum current for phase 2 [Amperes] [event] [Double]
	DYNAMIC_CIRCUIT_CURRENT_P3                         ObservationID = 113 // Dynamically set circuit maximum current for phase 3 [Amperes] [event] [Double]
	OUTPUT_CURRENT                                     ObservationID = 114 // Available current signaled to car with pilot tone [Double]
	DERATED_CURRENT                                    ObservationID = 115 // Available current after derating [A] [Double]
	DERATING_ACTIVE                                    ObservationID = 116 // Available current is limited by the charger due to high temperature [event] [Boolean]
	DEBUG_STRING                                       ObservationID = 117 // Debug string [String]
	ERROR_STRING                                       ObservationID = 118 // Descriptive error string [event] [String]
	ERROR_CODE                                         ObservationID = 119 // Error code according to error code table [event] [Integer]
	TOTAL_POWER                                        ObservationID = 120 // Total power [kW] [telemetry] [Double]
	SESSION_ENERGY                                     ObservationID = 121 // Session accumulated energy [kWh] [telemetry] [Double]
	ENERGY_PER_HOUR                                    ObservationID = 122 // Accumulated energy per hour [kWh] [event] [Double]
	LEGACY_EV_STATUS                                   ObservationID = 123 // 0 = not legacy ev, 1 = legacy ev detected, 2 = reviving ev [Integer]
	LIFETIME_ENERGY                                    ObservationID = 124 // Accumulated energy in the lifetime of the charger [kWh] [Double]
	LIFETIME_RELAY_SWITCHES                            ObservationID = 125 // Total number of relay switches in the lifetime of the charger (irrespective of the number of phases used) [Integer]
	LIFETIME_HOURS                                     ObservationID = 126 // Total number of hours in operation [Integer]
	DYNAMIC_CURRENT_OFFLINE_FALLBACK_DEPRICATED        ObservationID = 127 // Maximum circuit current when offline [event] [Integer]
	USER_IDTOKEN                                       ObservationID = 128 // User ID token string from RFID reading [event] [String]
	CHARGING_SESSION                                   ObservationID = 129 // Charging sessions [json] [event] [String]
	CELL_RSSI                                          ObservationID = 130 // Cellular signal strength [dBm] [telemetry] [Integer]
	CELL_RAT                                           ObservationID = 131 // Cellular radio access technology according to RAT table [event] [Integer]
	WI_FI_RSSI                                         ObservationID = 132 // WiFi signal strength [dBm] [telemetry] [Integer]
	CELL_ADDRESS                                       ObservationID = 133 // IP address assigned by cellular network [debug] [String]
	WI_FI_ADDRESS                                      ObservationID = 134 // IP address assigned by WiFi network [debug] [String]
	WI_FI_TYPE                                         ObservationID = 135 // WiFi network type letters (G, N, AC, etc) [debug] [String]
	LOCAL_RSSI                                         ObservationID = 136 // Local radio signal strength [dBm] [telemetry] [Integer]
	MASTER_BACK_PLATE_ID                               ObservationID = 137 // Back Plate RFID of master [event] [String]
	LOCAL_TX_POWER                                     ObservationID = 138 // Local radio transmission power [dBm] [telemetry] [Integer]
	LOCAL_STATE                                        ObservationID = 139 // Local radio state [event] [String]
	FOUND_WI_FI                                        ObservationID = 140 // List of found WiFi SSID and RSSI values [event] [String]
	CHARGER_RAT                                        ObservationID = 141 // Radio access technology in use: 0 = cellular, 1 = wifi [Integer]
	CELLULAR_INTERFACE_ERROR_COUNT                     ObservationID = 142 // The number of times since boot the system has reported an error on this interface [poll] [Integer]
	CELLULAR_INTERFACE_RESET_COUNT                     ObservationID = 143 // The number of times since boot the interface was reset due to high error count [poll] [Integer]
	WIFI_INTERFACE_ERROR_COUNT                         ObservationID = 144 // The number of times since boot the system has reported an error on this interface [poll] [Integer]
	WIFI_INTERFACE_RESET_COUNT                         ObservationID = 145 // The number of times since boot the interface was reset due to high error count [poll] [Integer]
	LOCAL_NODE_TYPE                                    ObservationID = 146 // 0-Unconfigured, 1-Master, 2-Extender, 3-End device [Integer]
	LOCAL_RADIO_CHANNEL                                ObservationID = 147 // Channel nr 0 - 11 [Integer]
	LOCAL_SHORT_ADDRESS                                ObservationID = 148 // Address of charger on local radio network [Integer]
	LOCAL_PARENT_ADDR_OR_NUM_OF_NODES                  ObservationID = 149 // If master-Number of slaves connected, If slave- Address parent [Integer]
	TEMP_MAX                                           ObservationID = 150 // Maximum temperature for all sensors [Celsius] [telemetry] [Double]
	TEMP_AMBIENT_POWER_BOARD                           ObservationID = 151 // Temperature measured by ambient sensor on power board [Celsius] [event] [Double]
	TEMP_INPUT_T2                                      ObservationID = 152 // Temperature at input T2 [Celsius] [event] [Double]
	TEMP_INPUT_T3                                      ObservationID = 153 // Temperature at input T3 [Celsius] [event] [Double]
	TEMP_INPUT_T4                                      ObservationID = 154 // Temperature at input T4 [Celsius] [event] [Double]
	TEMP_INPUT_T5                                      ObservationID = 155 // Temperature at input T5 [Celsius] [event] [Double]
	TEMP_OUTPUT_N                                      ObservationID = 160 // Temperature at type 2 connector plug for N [Celsius] [event] [Double]
	TEMP_OUTPUT_L1                                     ObservationID = 161 // Temperature at type 2 connector plug for L1 [Celsius] [event] [Double]
	TEMP_OUTPUT_L2                                     ObservationID = 162 // Temperature at type 2 connector plug for L2 [Celsius] [event] [Double]
	TEMP_OUTPUT_L3                                     ObservationID = 163 // Temperature at type 2 connector plug for L3 [Celsius] [event] [Double]
	TEMP_AMBIENT                                       ObservationID = 170 // Ambient temperature [Celsius] [event] [Double]
	LIGHT_AMBIENT                                      ObservationID = 171 // Ambient light from front side [Percent] [debug] [Integer]
	INT_REL_HUMIDITY                                   ObservationID = 172 // Internal relative humidity [Percent] [event] [Integer]
	BACK_PLATE_LOCKED                                  ObservationID = 173 // Back plate confirmed locked [event] [Boolean]
	CURRENT_MOTOR                                      ObservationID = 174 // Motor current draw [debug] [Double]
	BACK_PLATE_HALL_SENSOR                             ObservationID = 175 // Raw sensor value [mV] [Integer]
	IN_CURRENT_T2                                      ObservationID = 182 // Calculated current RMS for input T2 [Amperes] [telemetry] [Double]
	IN_CURRENT_T3                                      ObservationID = 183 // Current RMS for input T3 [Amperes] [telemetry] [Double]
	IN_CURRENT_T4                                      ObservationID = 184 // Current RMS for input T4 [Amperes] [telemetry] [Double]
	IN_CURRENT_T5                                      ObservationID = 185 // Current RMS for input T5 [Amperes] [telemetry] [Double]
	IN_VOLT_T1_T2                                      ObservationID = 190 // Input voltage RMS between T1 and T2 [Volt] [telemetry] [Double]
	IN_VOLT_T1_T3                                      ObservationID = 191 // Input voltage RMS between T1 and T3 [Volt] [telemetry] [Double]
	IN_VOLT_T1_T4                                      ObservationID = 192 // Input voltage RMS between T1 and T4 [Volt] [telemetry] [Double]
	IN_VOLT_T1_T5                                      ObservationID = 193 // Input voltage RMS between T1 and T5 [Volt] [telemetry] [Double]
	IN_VOLT_T2_T3                                      ObservationID = 194 // Input voltage RMS between T2 and T3 [Volt] [telemetry] [Double]
	IN_VOLT_T2_T4                                      ObservationID = 195 // Input voltage RMS between T2 and T4 [Volt] [telemetry] [Double]
	IN_VOLT_T2_T5                                      ObservationID = 196 // Input voltage RMS between T2 and T5 [Volt] [telemetry] [Double]
	IN_VOLT_T3_T4                                      ObservationID = 197 // Input voltage RMS between T3 and T4 [Volt] [telemetry] [Double]
	IN_VOLT_T3_T5                                      ObservationID = 198 // Input voltage RMS between T3 and T5 [Volt] [telemetry] [Double]
	IN_VOLT_T4_T5                                      ObservationID = 199 // Input voltage RMS between T4 and T5 [Volt] [telemetry] [Double]
	OUT_VOLT_PIN1_2                                    ObservationID = 202 // Output voltage RMS between type 2 pin 1 and 2 [Volt] [telemetry] [Double]
	OUT_VOLT_PIN1_3                                    ObservationID = 203 // Output voltage RMS between type 2 pin 1 and 3 [Volt] [telemetry] [Double]
	OUT_VOLT_PIN1_4                                    ObservationID = 204 // Output voltage RMS between type 2 pin 1 and 4 [Volt] [telemetry] [Double]
	OUT_VOLT_PIN1_5                                    ObservationID = 205 // Output voltage RMS between type 2 pin 1 and 5 [Volt] [telemetry] [Double]
	VOLT_LEVEL33                                       ObservationID = 210 // 3.3 Volt Level [Volt] [telemetry] [Double]
	VOLT_LEVEL5                                        ObservationID = 211 // 5 Volt Level [Volt] [telemetry] [Double]
	VOLT_LEVEL12                                       ObservationID = 212 // 12 Volt Level [Volt] [telemetry] [Double]
	LTE_RSRP                                           ObservationID = 220 // Reference Signal Received Power (LTE) [-144 .. -44 dBm] [Integer]
	LTE_SINR                                           ObservationID = 221 // Signal to Interference plus Noise Ratio (LTE) [-20 .. +30 dB] [Integer]
	LTE_RSRQ                                           ObservationID = 222 // Reference Signal Received Quality (LTE) [-19 .. -3 dB] [Integer]
	EQ_AVAILABLE_CURRENT_P1                            ObservationID = 230 // Available current for charging on P1 according to Equalizer [Double]
	EQ_AVAILABLE_CURRENT_P2                            ObservationID = 231 // Available current for charging on P2 according to Equalizer [Double]
	EQ_AVAILABLE_CURRENT_P3                            ObservationID = 232 // Available current for charging on P3 according to Equalizer [Double]
	LISTEN_TO_CONTROL_PULSE                            ObservationID = 56  // True = charger needs control pulse to consider itself online. Readback on charger setting [event] [Boolean]
	CONTROL_PULSE_RTT                                  ObservationID = 57  // Control pulse round-trip time in milliseconds [Integer]
)
