package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util/redact"
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
	clientsecret: geheim
	token:
		accesstoken: geheim
		refreshtoken: geheim
	pin: geheim
	mac: geheim
	clientsecret: geheim # comment
	clientsecret : geheim
	`

	if res := redact.String(secret); strings.Contains(res, "geheim") || !strings.Contains(res, "public") {
		t.Errorf("secret exposed: %v", res)
	}
}
