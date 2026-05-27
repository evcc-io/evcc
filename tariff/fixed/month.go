package fixed

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

//go:generate go tool enumer -type Month
type Month int

const (
	January Month = iota
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

var Year = []Month{January, February, March, April, May, June, July, August, September, October, November, December}

var shortMonths = map[string]Month{
	// english
	"jan": January,
	"feb": February,
	"mar": March,
	"apr": April,
	"may": May,
	"jun": June,
	"jul": July,
	"aug": August,
	"sep": September,
	"oct": October,
	"nov": November,
	"dec": December,
	// german
	// "jan": January,
	// "feb": February,
	"mär": March,
	// "apr": April,
	"mai": May,
	// "jun": June,
	// "jul": July,
	// "aug": August,
	// "sep": September,
	"okt": October,
	// "nov": November,
	"dez": December,
}

// ParseMonth parses a single month
func ParseMonth(s string) (Month, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	// full string
	if m, err := MonthString(s); err == nil {
		return m, nil
	}

	// short string
	if m, ok := shortMonths[s]; ok {
		return m, nil
	}

	m, err := strconv.Atoi(s)
	if m < 1 || m > 12 || err != nil {
		return 0, fmt.Errorf("invalid month: %s", s)
	}

	return Month(m - 1), nil
}

// ParseMonths converts a months string into a slice of individual months
// Months format:
//
//	month[-month][, ...]
func ParseMonths(s string) ([]Month, error) {
	var res []Month

	for segment := range strings.SplitSeq(s, ",") {
		fromto := strings.SplitN(segment, "-", 2)
		if len(fromto) == 0 {
			return nil, fmt.Errorf("invalid month range: %s", segment)
		}

		fromToFrom := fromto[0]

		// single empty segment
		if len(fromto) == 1 && strings.TrimSpace(fromToFrom) == "" {
			return slices.Clone(Year), nil
		}

		from, err := ParseMonth(fromToFrom)
		if err != nil {
			return nil, err
		}
		res = append(res, from)

		if len(fromto) == 2 {
			to, err := ParseMonth(fromto[1])
			if err != nil {
				return nil, err
			}

			if to < from {
				to += 12
			}

			for m := from + 1; m <= to; m++ {
				res = append(res, m%12)
			}
		}
	}

	if len(res) > 12 {
		return nil, errors.New("too many months")
	}

	if len(slices.Compact(slices.Sorted(slices.Values(res)))) < len(res) {
		return nil, errors.New("duplicate months")
	}

	return res, nil
}
