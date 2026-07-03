package hems

import "github.com/evcc-io/evcc/api"

// API describes the HEMS system interface combining the runtime loop
// with the api.HEMS state-query surface consumed by Circuit, Site and Loadpoint.
type API interface {
	api.HEMS
	Run()
}
