package tariff

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/provider/golang"
	"github.com/evcc-io/evcc/provider/golang/stdlib"
	"github.com/traefik/yaegi/interp"
)

type embed struct {
	Charges float64 `mapstructure:"charges"`
	Tax     float64 `mapstructure:"tax"`
	Formula string  `mapstructure:"formula"`

	calc func(float64, time.Time) (float64, error)
}

func (t *embed) init() (err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	if t.Formula == "" {
		return nil
	}

	vm := interp.New(interp.Options{})
	if err := vm.Use(stdlib.Symbols); err != nil {
		return err
	}

	if _, err := vm.Eval(fmt.Sprintf(`%s
	var (
		price float64
		charges float64 = %f
		tax float64 = %f
		ts time.Time
	)`, golang.Imports, t.Charges, t.Tax)); err != nil {
		return err
	}

	prg, err := vm.Compile(t.Formula)
	if err != nil {
		return err
	}

	t.calc = func(price float64, ts time.Time) (float64, error) {
		if _, err := vm.Eval(fmt.Sprintf(`
			price = %f
			ts = time.Unix(%d, 0).Local()
		`, price, ts.Unix())); err != nil {
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
	_, err = t.calc(0, time.Now())

	return err
}

func (t *embed) totalPrice(price float64, ts time.Time) float64 {
	if t.calc != nil {
		res, _ := t.calc(price, ts)
		return res
	}
	return (price + t.Charges) * (1 + t.Tax)
}
