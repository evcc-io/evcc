package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/evcc-io/evcc/util"
)

type waitForResponseProvider struct {
	ctx  context.Context
	set  Config
	get  Config
	done chan getterResult
	log  *util.Logger
}

type getterResult struct {
	val string
	err error
}

func init() {
	registry.AddCtx("waitForResponse", NewWaitForResponseFromConfig)
}

func NewWaitForResponseFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Set Config
		Get Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("waitForResponse")

	wfr := &waitForResponseProvider{
		ctx:  ctx,
		set:  cc.Set,
		get:  cc.Get,
		log:  log,
		done: make(chan getterResult),
	}

	return wfr, nil
}

func (wfr *waitForResponseProvider) IntSetter(param string) (func(int64) error, error) {
	return func(val int64) error {
		setter, err := NewIntSetterFromConfig(wfr.ctx, param, wfr.set)
		if err != nil {
			return err
		}

		getter, err := NewStringGetterFromConfig(wfr.ctx, wfr.get)
		if err != nil {
			return err
		}

		go func() {
			val, err := getter()
			wfr.log.DEBUG.Println("got", val, err)
			wfr.done <- getterResult{val, err}
		}()
		wfr.log.DEBUG.Println("setting", val)
		setter(val)
		result := <-wfr.done
		if result.err != nil {
			return fmt.Errorf("timeout waiting for response")
		}

		success, err := strconv.ParseBool(result.val)
		if err != nil {
			return err
		}
		if !success {
			return fmt.Errorf("failed to set value: %s", result.val)
		}
		return nil
	}, nil
}
