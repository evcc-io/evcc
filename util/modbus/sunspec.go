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
func ParsePoint(selector string) ([]SunSpecOperation, error) {
	el := strings.Split(selector, ":")
	if len(el) < 2 || len(el) > 3 {
		return nil, fmt.Errorf("invalid sunspec format: %s", selector)
	}

	models := strings.Split(el[0], "|")
	if len(models) == 0 {
		return nil, fmt.Errorf("missing sunspec model: %s", selector)
	}

	var res []SunSpecOperation
	for _, m := range models {
		model, err := strconv.Atoi(m)
		if err != nil {
			return nil, fmt.Errorf("invalid sunspec model: %s", selector)
		}

		var block int
		if len(el) == 3 {
			// block is the middle element
			block, err = strconv.Atoi(el[1])
			if err != nil {
				return nil, fmt.Errorf("invalid sunspec block: %s", selector)
			}
		}

		res = append(res, SunSpecOperation{
			Model: model,
			Block: block,
			Point: el[len(el)-1],
		})
	}

	return res, nil
}
