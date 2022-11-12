package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestUnwrap(t *testing.T) {
	err := fmt.Errorf("foo: %w", fmt.Errorf("bar %w", errors.New("baz")))

	res := unwrap(err)
	if exp := []string{"foo", "bar", "baz"}; !reflect.DeepEqual(res, exp) {
		t.Errorf("expected %v, got %v", exp, res)
	}
}

func TestRedact(t *testing.T) {
	secret := `
	# sponsor token is a public token
	sponsortoken: geheim
	user: geheim
	password: geheim
	secret: geheim
	token:
		access: geheim
		refresh: geheim
	pin: geheim
	mac: geheim
	secret: geheim # comment
	secret : geheim
	`

	if res := redact(secret); strings.Contains(res, "geheim") || !strings.Contains(res, "public") {
		t.Errorf("secret exposed: %v", res)
	}
}
