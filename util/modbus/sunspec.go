package modbus

import (
	"fmt"
	"strconv"
	"strings"
)

// SunSpecOperation is a sunspec modbus operation
type SunSpecOperation struct {
	Model, Block int
	Point        string
}

// ParsePoint parses sunspec point from string
func ParsePoint(selector string) (SunSpecOperation, error) {
	var (
		res SunSpecOperation
		err error
	)

	el := strings.Split(selector, ":")
	if len(el) < 2 || len(el) > 3 {
		return res, fmt.Errorf("invalid sunspec format: %s", selector)
	}

	if res.Model, err = strconv.Atoi(el[0]); err != nil {
		return res, fmt.Errorf("invalid sunspec model: %s", selector)
	}

	if len(el) == 3 {
		// block is the middle element
		res.Block, err = strconv.Atoi(el[1])
		if err != nil {
			return res, fmt.Errorf("invalid sunspec block: %s", selector)
		}
	}

	res.Point = el[len(el)-1]

	return res, nil
}
