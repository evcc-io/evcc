package zaptec

const ApiURL = "https://api.zaptec.com"

type ObservationID int

//go:generate enumer -type ObservationID

// Commands
const (
	CmdStop   = 506
	CmdResume = 507
)

const (
	OpModeUnknown             = 0
	OpModeDisconnected        = 1
	OpModeConnectedRequesting = 2
	OpModeConnectedCharging   = 3
	OpModeConnectedFinished   = 5
)

// Observations
const (
	Unknown                                     ObservationID = 0
	OfflineMode                                 ObservationID = 1
	AuthenticationRequired                      ObservationID = 120
	PaymentActive                               ObservationID = 130
	PaymentCurrency                             ObservationID = 131
	PaymentSessionUnitPrice                     ObservationID = 132
	PaymentEnergyUnitPrice                      ObservationID = 133
	PaymentTimeUnitPrice                        ObservationID = 134
	CommunicationMode                           ObservationID = 150
	PermanentCableLock                          ObservationID = 151
	ProductCode                                 ObservationID = 152
	HmiBrightness                               ObservationID = 153
	LockCableWhenConnected                      ObservationID = 154
	SoftStartDisabled                           ObservationID = 155
	FirmwareApiHost                             ObservationID = 156
	MIDBlinkEnabled                             ObservationID = 170
	TemperatureInternal5                        ObservationID = 201
	TemperatureInternal6                        ObservationID = 202
	TemperatureInternalLimit                    ObservationID = 203
	TemperatureInternalMaxLimit                 ObservationID = 241
	Humidity                                    ObservationID = 270
	VoltagePhase1                               ObservationID = 501
	VoltagePhase2                               ObservationID = 502
	VoltagePhase3                               ObservationID = 503
	CurrentPhase1                               ObservationID = 507
	CurrentPhase2                               ObservationID = 508
	CurrentPhase3                               ObservationID = 509
	ChargerMaxCurrent                           ObservationID = 510
	ChargerMinCurrent                           ObservationID = 511
	ActivePhases                                ObservationID = 512
	TotalChargePower                            ObservationID = 513
	RcdCurrent                                  ObservationID = 515
	Internal12vCurrent                          ObservationID = 517
	PowerFactor                                 ObservationID = 518
	SetPhases                                   ObservationID = 519
	MaxPhases                                   ObservationID = 520
	ChargerOfflinePhase                         ObservationID = 522
	ChargerOfflineCurrent                       ObservationID = 523
	RcdCalibration                              ObservationID = 540
	RcdCalibrationNoise                         ObservationID = 541
	TotalChargePowerSession                     ObservationID = 553
	SignedMeterValue                            ObservationID = 554
	SignedMeterValueInterval                    ObservationID = 555
	SessionEnergyCountExportActive              ObservationID = 560
	SessionEnergyCountExportReactive            ObservationID = 561
	SessionEnergyCountImportActive              ObservationID = 562
	SessionEnergyCountImportReactive            ObservationID = 563
	SoftStartTime                               ObservationID = 570
	ChargeDuration                              ObservationID = 701
	ChargeMode                                  ObservationID = 702
	ChargePilotLevelInstant                     ObservationID = 703
	ChargePilotLevelAverage                     ObservationID = 704
	PilotVsProximityTime                        ObservationID = 706
	ChargeCurrentInstallationMaxLimit           ObservationID = 707
	ChargeCurrentSet                            ObservationID = 708
	ChargerOperationMode                        ObservationID = 710
	IsEnabled                                   ObservationID = 711
	IsStandAlone                                ObservationID = 712
	ChargerCurrentUserUuidDeprecated            ObservationID = 713
	CableType                                   ObservationID = 714
	NetworkType                                 ObservationID = 715
	DetectedCar                                 ObservationID = 716
	GridTestResult                              ObservationID = 717
	FinalStopActive                             ObservationID = 718
	SessionIdentifier                           ObservationID = 721
	ChargerCurrentUserUuid                      ObservationID = 722
	CompletedSession                            ObservationID = 723
	NewChargeCard                               ObservationID = 750
	AuthenticationListVersion                   ObservationID = 751
	EnabledNfcTechnologies                      ObservationID = 752
	LteRoamingDisabled                          ObservationID = 753
	InstallationId                              ObservationID = 800
	RoutingId                                   ObservationID = 801
	Notifications                               ObservationID = 803
	Warnings                                    ObservationID = 804
	DiagnosticsMode                             ObservationID = 805
	InternalDiagnosticsLog                      ObservationID = 807
	DiagnosticsString                           ObservationID = 808
	CommunicationSignalStrength                 ObservationID = 809
	CloudConnectionStatus                       ObservationID = 810
	McuResetSource                              ObservationID = 811
	McuRxErrors                                 ObservationID = 812
	McuToVariscitePacketErrors                  ObservationID = 813
	VarisciteToMcuPacketErrors                  ObservationID = 814
	UptimeVariscite                             ObservationID = 820
	UptimeMCU                                   ObservationID = 821
	CarSessionLog                               ObservationID = 850
	CommunicationModeConfigurationInconsistency ObservationID = 851
	RawPilotMonitor                             ObservationID = 852
	IT3PhaseDiagnosticsLog                      ObservationID = 853
	PilotTestResults                            ObservationID = 854
	UnconditionalNfcDetectionIndication         ObservationID = 855
	EmcTestCounter                              ObservationID = 899
	ProductionTestResults                       ObservationID = 900
	PostProductionTestResults                   ObservationID = 901
	SmartMainboardSoftwareApplicationVersion    ObservationID = 908
	SmartMainboardSoftwareBootloaderVersion     ObservationID = 909
	SmartComputerSoftwareApplicationVersion     ObservationID = 911
	SmartComputerSoftwareBootloaderVersion      ObservationID = 912
	SmartComputerHardwareVersion                ObservationID = 913
	MacMain                                     ObservationID = 950
	MacPlcModuleGrid                            ObservationID = 951
	MacWiFi                                     ObservationID = 952
	MacPlcModuleEv                              ObservationID = 953
	LteImsi                                     ObservationID = 960
	LteMsisdn                                   ObservationID = 961
	LteIccid                                    ObservationID = 962
	LteImei                                     ObservationID = 963
	MIDCalibration                              ObservationID = 980
	IsOcppConnected                             ObservationID = -3
	IsOnline                                    ObservationID = -2
	Pulse                                       ObservationID = -1
)
