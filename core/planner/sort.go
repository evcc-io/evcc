package planner

import "github.com/evcc-io/evcc/api"

// SortByTime is a sortFunc for slices.Sort
func SortByTime(i, j api.Rate) bool {
	return i.Start.Before(j.Start)
}

// sortByCost is a sortFunc for slices.Sort
func sortByCost(i, j api.Rate) bool {
	if i.Price == j.Price {
		return i.Start.After(j.Start)
	}

	return i.Price < j.Price
}
