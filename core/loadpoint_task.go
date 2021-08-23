package core

import (
	"errors"

	"github.com/evcc-io/evcc/api"
)

// task adds a task to the list of running tasks
func (lp *LoadPoint) task(task func() error) {
	lp.tasks = append(lp.tasks, task)
}

// runTasks runs all defined tasks
func (lp *LoadPoint) runTasks() {
	var incomplete []func() error
	for _, task := range lp.tasks {
		err := task()
		if errors.Is(err, api.ErrMustRetry) {
			incomplete = append(incomplete, task)
		}
	}
	lp.tasks = incomplete
}

func (lp *LoadPoint) odometer() error {
	v, ok := lp.vehicle.(api.VehicleOdometer)
	if !ok {
		return nil
	}

	odo, err := v.Odometer()
	switch err {
	case nil:
		lp.publish("vehicleOdometer", odo)
	case api.ErrMustRetry:
	default:
		lp.log.ERROR.Printf("vehicle odometer: %v", err)
	}
	return err
}
