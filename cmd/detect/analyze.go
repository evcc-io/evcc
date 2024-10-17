package detect

import "github.com/evcc-io/evcc/cmd/detect/tasks"

type Criteria map[string]interface{}

type TypeSummary struct {
	Results       []tasks.Result
	Found, Unique bool
}

type Summary struct {
	Charger, Grid, PV, Charge, Battery, Meter TypeSummary
}
