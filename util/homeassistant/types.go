package homeassistant

type StateResponse struct {
	EntityId   string `json:"entity_id"`
	State      string `json:"state"`
	Attributes struct {
		UnitOfMeasurement string `json:"unit_of_measurement"`
		DeviceClass       string `json:"device_class"`
		FriendlyName      string `json:"friendly_name"`
	} `json:"attributes"`
}
