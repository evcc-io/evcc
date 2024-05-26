package connect

const StatusSuccess = "SUCCESS"

type VehiclesResponse struct {
	Vehicles []Vehicle
}

type Vehicle struct {
	VehicleID                     string `json:"vehicleId"`
	Make                          string `json:"make"`
	ModelName                     string `json:"modelName"`
	ModelYear                     string `json:"modelYear"`
	Color                         string `json:"color"`
	NickName                      string `json:"nickName"`
	LastUpdated                   string `json:"lastUpdated"`
	VehicleAuthorizationIndicator int    `json:"vehicleAuthorizationIndicator"`
	ServiceCompatible             bool   `json:"serviceCompatible"`
	EngineType                    string `json:"engineType"`
}

type InformationResponse struct {
	Status  string
	Vehicle Vehicle
}
