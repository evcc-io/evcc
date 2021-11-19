package ford

import "time"

const (
	AuthURI        = "https://sso.ci.ford.ca" // fcis.ice.ibmcloud.com
	ApiURI         = "https://usapi.cv.ford.com"
	VehicleListURI = "https://api.mps.ford.com/api/users/vehicles"
	RefreshTimeout = time.Minute           // timeout to get status after refresh
	TimeFormat     = "01-02-2006 15:04:05" // time format used by Ford API, time is in UTC
)
