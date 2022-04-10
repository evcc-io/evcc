package zaptec

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

type State struct {
	ChargerId     string
	StateId       int
	Timestamp     string
	ValueAsString string
}
