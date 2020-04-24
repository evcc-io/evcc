package core

import (
	"fmt"
	"sync"
)

type Cache struct {
	sync.Mutex
	values map[string]map[string]interface{}
}

func (c *Cache) All(loadPoint string) (map[string]interface{}, error) {
	c.Lock()
	defer c.Unlock()

	lp, ok := c.values[loadPoint]
	if !ok {
		return nil, fmt.Errorf("invalid loadpoint: %s", loadPoint)
	}

	return lp, nil
}

func (c *Cache) Run(in <-chan Param) {
	c.values = make(map[string]map[string]interface{})

	for param := range in {
		c.Lock()

		lp, ok := c.values[param.LoadPoint]
		if !ok {
			lp = make(map[string]interface{})
			c.values[param.LoadPoint] = lp
		}

		lp[param.Key] = param.Val

		c.Unlock()
	}
}
