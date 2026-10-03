package tuya

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// Diagnose prints all data points reported by the device with their code names, ordered by id
func Diagnose(dps map[string]any, names map[string]string) {
	ids := slices.SortedFunc(maps.Keys(dps), func(a, b string) int {
		i, _ := strconv.Atoi(a)
		j, _ := strconv.Atoi(b)
		return i - j
	})

	for _, id := range ids {
		fmt.Printf("\t%s %s:\t%v\n", id, names[id], dps[id])
	}
}
