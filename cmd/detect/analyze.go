package detect

import "github.com/evcc-io/evcc/cmd/detect/tasks"

type Criteria map[string]interface{}

func filter(list []tasks.Result, criteria []Criteria) (match []tasks.Result) {
	for _, res := range list {
		for _, criterium := range criteria {
			ok := true

			for matchKey, matchVal := range criterium {
				if foundVal, found := res.Attributes[matchKey]; !found || foundVal != matchVal {
					ok = false
					break
				}
			}

			if ok {
				match = append(match, res)
			}
		}
	}

	return match
}

type TypeSummary struct {
	Results       []tasks.Result
	Found, Unique bool
}

type Summary struct {
	Charger, Grid, PV, Charge, Battery, Meter TypeSummary
}

func summarize(res []tasks.Result) TypeSummary {
	return TypeSummary{
		Results: res,
		Found:   len(res) > 0,
		Unique:  len(res) == 1,
	}
}

const (
	tid     = "task.id"
	smaHttp = "details.http"
)

func Consolidate(res []tasks.Result) Summary {
	grid := filter(res, []Criteria{
		{tid: taskOpenwb},
		{tid: taskSMA, smaHttp: false},
		{tid: taskE3DC},
		{tid: taskSonnen},
	})

	pv := filter(res, []Criteria{
		{tid: taskOpenwb},
		{tid: taskInverter},
	})

	battery := filter(res, []Criteria{
		{tid: taskOpenwb},
		{tid: taskE3DC},
		{tid: taskSonnen},
		{tid: taskBattery},
	})

	charger := filter(res, []Criteria{
		{tid: taskOpenwb},
		{tid: taskWallbe},
		{tid: taskPhoenixEMEth},
		{tid: taskPhoenixEVEth},
		{tid: taskEVSEWifi},
		{tid: taskGoE},
		{tid: taskKEBA},
	})

	charge := filter(res, []Criteria{
		{tid: taskOpenwb},
		{tid: taskKEBA},
	})

	meter := filter(res, []Criteria{
		{tid: taskSMA, smaHttp: true},
		{tid: taskMeter},
	})

	return Summary{
		Grid:    summarize(grid),
		PV:      summarize(pv),
		Battery: summarize(battery),
		Charger: summarize(charger),
		Charge:  summarize(charge),
		Meter:   summarize(meter),
	}
}
