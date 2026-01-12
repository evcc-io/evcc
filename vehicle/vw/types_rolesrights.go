package vw

const StatusService = "statusreport_v1"

// RolesRights is the /rolesrights/operationlist response
type RolesRights struct {
	OperationList struct {
		VIN, UserId, Role, Status string
		ServiceInfo               []ServiceInfo
	}
}

func (rr RolesRights) ServiceByID(id string) *ServiceInfo {
	for _, s := range rr.OperationList.ServiceInfo {
		if s.ServiceId == id {
			return &s
		}
	}
	return nil
}

// ServiceInfo is the rolesrights service information
type ServiceInfo struct {
	ServiceId     string
	ServiceType   string
	ServiceStatus struct {
		Status string
	}
	LicenseRequired            bool
	CumulatedLicense           map[string]any
	PrimaryUserRequired        bool
	TermsAndConditionsRequired bool
	ServiceEol                 string
	RolesAndRightsRequired     bool
	InvocationUrl              struct {
		Content string
	}
	Operation []map[string]any
}
