package planner

import "github.com/evcc-io/evcc/api"

// SortByTime is a sortFunc for slices.Sort
func SortByTime(i, j api.Rate) int {
	return i.Start.Compare(j.Start)
}

// sortByCost is a sortFunc for slices.Sort
func sortByCost(i, j api.Rate) int {
	switch {
	case i.Price < j.Price:
		return -1
	case i.Price > j.Price:
		return +1
	default:
		return j.Start.Compare(i.Start)
	}
}
