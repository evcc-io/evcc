package shelly

import "fmt"

func apiCall[T any](c *gen2Conn, api string) func() (T, error) {
	return func() (T, error) {
		var res T
		if err := c.execCmd(fmt.Sprintf("%s?id=%d", api, c.channel), false, &res); err != nil {
			return res, err
		}
		return res, nil
	}
}
