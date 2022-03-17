package silence

const ApiUri = "https://api.connectivity.silence.eco/api/v1/me/scooters?details=true&dynamic=true"

type Vehicle struct {
	ID           string
	Model        string
	Name         string
	FrameNo      string
	BatteryOut   bool
	Charging     bool
	LastLocation struct {
		Latitude     float64
		Longitude    float64
		Altitude     int
		CurrentSpeed int
		Time         string
	}
	BatterySoc          int
	Odometer            int
	BatteryTemperature  int
	MotorTemperature    int
	InverterTemperature int
	Range               int
	Velocity            int
	Status              int
	LastReportTime      string
	LastConnection      string
}
