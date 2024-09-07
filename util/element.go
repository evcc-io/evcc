package util

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// EnsureElementEx extracts objects with matching ID from list of objects
func EnsureElementEx[T any](
	id string,
	list func() ([]T, error),
	extract func(T) (string, error),
) (T, error) {
	var zero T

	objects, err := list()
	if err != nil {
		return zero, fmt.Errorf("cannot get Objects: %w", err)
	}

	if oin := strings.ToUpper(id); oin != "" {
		// oin defined
		for _, object := range objects {
			vv, err := extract(object)
			if err != nil {
				return zero, err
			}
			if strings.ToUpper(vv) == oin {
				return object, nil
			}
		}
	} else if len(objects) == 1 {
		// vin empty and exactly one vehicle
		return objects[0], nil
	}

	return zero, fmt.Errorf("cannot find element, got: %v", lo.Map(objects, func(v T, _ int) string {
		oin, _ := extract(v)
		return oin
	}))
}
