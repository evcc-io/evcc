package detect

import (
	"sort"
	"strings"

	"github.com/fatih/structs"
	"github.com/jeremywohl/flatten"
)

type TypeSummary struct {
	Results       []Result
	Found, Unique bool
}

type Summary struct {
	Charger, Grid, PV, Charge, Battery, Meter TypeSummary
}

type Criteria map[string]interface{}

// Prepare results
func Prepare(res []Result) []Result {
	for idx, hit := range res {
		if sma, ok := hit.Details.(SmaResult); ok {
			hit.Host = sma.Addr
		}

		hit.Attributes = make(map[string]interface{})
		flat, _ := flatten.Flatten(structs.Map(hit), "", flatten.DotStyle)
		for k, v := range flat {
			hit.Attributes[strings.ToLower(k)] = v
		}
		// fmt.Println(hit.Attributes)

		res[idx] = hit
	}

	// sort by host
	sort.Slice(res, func(i, j int) bool { return res[i].Host < res[j].Host })

	return res
}

func filter(list []Result, criteria []Criteria) (match []Result) {
	for _, res := range list {
		for _, matcher := range criteria {
			ok := true

			for k, matchVal := range matcher {
				if foundVal, found := res.Attributes[k]; !found || foundVal != matchVal {
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

func summarize(res []Result) TypeSummary {
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

func Consolidate(res []Result) Summary {
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
		{tid: taskPhoenixEMCP},
		{tid: taskEVSEWifi},
		{tid: taskGoE},
	})

	charge := filter(res, []Criteria{
		{tid: taskOpenwb},
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
