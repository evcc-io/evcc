package main

import (
	"fmt"
	"testing"

	"github.com/evcc-io/evcc/templates/builtin"
	"github.com/evcc-io/evcc/vehicle"
)

// TODO delete file

func TestGen(t *testing.T) {
	for typ, m := range vehicle.Registry {
		if typ != "audi" {
			continue
		}

		fmt.Println("---")
		fmt.Println(typ)

		if m.Config == nil {
			continue
		}

		foo := builtin.Annotate(m.Config)
		fmt.Println(foo)
	}
}
