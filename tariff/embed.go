package tariff

import (
	"errors"
	"fmt"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type embed struct {
	Charges float64 `mapstructure:"charges"`
	Tax     float64 `mapstructure:"tax"`
	Formula string  `mapstructure:"formula"`

	calc func(float64) (float64, error)
}

func (t *embed) init() error {
	if t.Formula == "" {
		return nil
	}

	vm := interp.New(interp.Options{})
	if err := vm.Use(stdlib.Symbols); err != nil {
		return err
	}

	if _, err := vm.Eval(`import "math"`); err != nil {
		return err
	}

	if _, err := vm.Eval("var price, charges, tax float64"); err != nil {
		return err
	}

	if _, err := vm.Eval(fmt.Sprintf("charges = %f", t.Charges)); err != nil {
		return err
	}

	if _, err := vm.Eval(fmt.Sprintf("tax = %f", t.Tax)); err != nil {
		return err
	}

	prg, err := vm.Compile(t.Formula)
	if err != nil {
		return err
	}

	t.calc = func(price float64) (float64, error) {
		if _, err := vm.Eval(fmt.Sprintf("price = %f", price)); err != nil {
			return 0, err
		}

		res, err := vm.Execute(prg)
		if err != nil {
			return 0, err
		}

		if !res.CanFloat() {
			return 0, errors.New("formula did not return a float value")
		}

		return res.Float(), nil
	}

	// test the formula
	_, err = t.calc(0)

	return err
}

func (t *embed) totalPrice(price float64) float64 {
	if t.calc != nil {
		res, _ := t.calc(price)
		return res
	}
	return (price + t.Charges) * (1 + t.Tax)
}
