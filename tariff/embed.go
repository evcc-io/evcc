package tariff

import (
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/plugin/golang/stdlib"
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

	t.calc = func(price float64, ts time.Time) (float64, error) {
		vm := interp.New(interp.Options{})
		if err := vm.Use(stdlib.Symbols); err != nil {
			return 0, err
		}
		vm.ImportUsed()

		if _, err := vm.Eval(fmt.Sprintf(`
		var (
			price float64 = %f
			charges float64 = %f
			tax float64 = %f
			ts = time.Unix(%d, 0).Local()
		)`, price, t.Charges, t.Tax, ts.Unix())); err != nil {
			return 0, err
		}

		res, err := vm.Eval(t.Formula)
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
