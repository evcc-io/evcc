package planner

import "github.com/evcc-io/evcc/api"

// sortByCost is a sortFunc for slices.Sort
func sortByCost(i, j api.Rate) int {
	switch {
	case i.Value < j.Value:
		return -1
	case i.Value > j.Value:
		return +1
	default:
		return j.Start.Compare(i.Start)
	}
}
