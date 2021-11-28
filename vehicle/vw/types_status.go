package vw

const ServiceOdometer = "0x0101010002"

type StatusResponse struct {
	StoredVehicleDataResponse struct {
		VIN         string
		VehicleData struct {
			Data []ServiceDefinition
		}
	}
	Error *Error
}

func (s *StatusResponse) ServiceByID(id string) *ServiceDefinition {
	for _, d := range s.StoredVehicleDataResponse.VehicleData.Data {
		if d.ID == id {
			return &d
		}
	}

	return nil
}

type ServiceDefinition struct {
	ID    string
	Field []FieldDefinition
}

func (s *ServiceDefinition) FieldByID(id string) *FieldDefinition {
	if s == nil {
		return nil
	}

	for _, f := range s.Field {
		if f.ID == id {
			return &f
		}
	}

	return nil
}

type FieldDefinition struct {
	ID               string // "0x0101010001",
	TsCarSentUtc     string // "2021-09-05T07:54:20Z",
	TsCarSent        string // "2021-09-05T07:54:19",
	TsCarCaptured    string // "2021-09-05T07:54:19",
	TsTssReceivedUtc string // "2021-09-05T07:54:23Z",
	MilCarCaptured   int    // 25009,
	MilCarSent       int    // 25009,
	Value            string // "echo"
}
