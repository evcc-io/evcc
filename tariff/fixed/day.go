package fixed

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

//go:generate enumer -type Day
type Day int

const (
	Sunday Day = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

var Week = []Day{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday}

var shortDays = map[string]Day{
	// english
	"sun": Sunday,
	"mon": Monday,
	"tue": Tuesday,
	"wed": Wednesday,
	"thu": Thursday,
	"fri": Friday,
	"sat": Saturday,
	// german
	"so": Sunday,
	"mo": Monday,
	"di": Tuesday,
	"mi": Wednesday,
	"do": Thursday,
	"fr": Friday,
	"sa": Saturday,
}

// ParseDay parses a single day
func ParseDay(s string) (Day, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	// full string
	if d, err := DayString(s); err == nil {
		return d, nil
	}

	// short string
	if d, ok := shortDays[s]; ok {
		return d, nil
	}

	d, err := strconv.Atoi(s)
	if d < 0 || d > 7 || err != nil {
		return 0, fmt.Errorf("invalid day: %s", s)
	}

	return Day(d % 7), nil
}

// ParseDays converts a days string into a slice of individual days
// Days format:
//
//	day[-day][, ...]
func ParseDays(s string) ([]Day, error) {
	var res []Day

	for _, segment := range strings.Split(s, ",") {
		fromto := strings.SplitN(segment, "-", 2)
		if len(fromto) == 0 {
			return nil, fmt.Errorf("invalid day range: %s", segment)
		}

		from, err := ParseDay(fromto[0])
		if err != nil {
			return nil, err
		}
		res = append(res, Day(from%7))

		if len(fromto) == 2 {
			to, err := ParseDay(fromto[1])
			if err != nil {
				return nil, err
			}

			if to < from {
				to += 7
			}

			for d := from + 1; d <= to; d++ {
				res = append(res, d%7)
			}
		}
	}

	if len(res) > 7 {
		return nil, errors.New("too many days")
	}

	sorted := make([]Day, len(res))
	copy(sorted, res)
	slices.Sort(sorted)

	if len(slices.Compact(sorted)) < len(res) {
		return nil, errors.New("duplicate days")
	}

	return res, nil
}
