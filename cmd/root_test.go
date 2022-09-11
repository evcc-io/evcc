package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestUnwrap(t *testing.T) {
	err := fmt.Errorf("foo: %w", fmt.Errorf("bar %w", errors.New("baz")))

	res := unwrap(err)
	if exp := []string{"foo", "bar", "baz"}; !reflect.DeepEqual(res, exp) {
		t.Errorf("expected %v, got %v", exp, res)
	}
}
