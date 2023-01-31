package fixed

import (
	"fmt"
	"strings"
	"time"
)

type HourMin struct {
	Hour, Min int
}

func (hm HourMin) Minutes() int {
	return 60*hm.Hour + hm.Min
}

func (hm HourMin) IsNil() bool {
	return hm.Hour+hm.Min == 0
}

func (hm HourMin) String() string {
	return fmt.Sprintf("%02d:%02d", hm.Hour, hm.Min)
}

type TimeRange struct {
	From, To HourMin
}

func (tr TimeRange) Contains(hm HourMin) bool {
	ts := hm.Minutes()
	return tr.From.Minutes() <= ts && (tr.To.IsNil() || tr.To.Minutes() > ts)
}

func (tr TimeRange) IsNil() bool {
	return tr.From.IsNil() && tr.To.IsNil()
}

func (tr TimeRange) String() string {
	to := tr.To.String()
	if tr.To.IsNil() {
		to += "(+1)"
	}
	return tr.From.String() + "-" + to
}

func parseTime(s string) (HourMin, error) {
	s = strings.TrimSpace(s)

	t, err := time.ParseInLocation("15:04", s, time.Local)
	if err == nil {
		return HourMin{t.Hour(), t.Minute()}, nil
	}

	t, err = time.ParseInLocation("15", s, time.Local)
	if err == nil {
		return HourMin{t.Hour(), 0}, nil
	}

	return HourMin{}, fmt.Errorf("invalid time: %s", s)
}

func ParseTimeRange(s string) (TimeRange, error) {
	fromto := strings.SplitN(s, "-", 2)
	if len(fromto) != 2 {
		return TimeRange{}, fmt.Errorf("invalid time range: %s", s)
	}

	from, err := parseTime(fromto[0])
	if err != nil {
		return TimeRange{}, err
	}

	to, err := parseTime(fromto[1])
	if err != nil {
		return TimeRange{}, err
	}

	if tom := to.Minutes(); tom != 0 && from.Minutes() >= tom {
		return TimeRange{}, fmt.Errorf("invalid time range: %s, <from> must be before <to>", s)
	}

	return TimeRange{from, to}, nil
}

func ParseTimeRanges(s string) ([]TimeRange, error) {
	var res []TimeRange

	for _, segment := range strings.Split(s, ",") {
		tr, err := ParseTimeRange(segment)
		if err != nil {
			return nil, err
		}
		res = append(res, tr)
	}

	return res, nil
}
