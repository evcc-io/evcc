package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/redact"
	"github.com/stretchr/testify/assert"
)

func TestCircuitsSource(t *testing.T) {
	yamlSource.circuits = globalconfig.YamlSourceFile
	t.Cleanup(func() { yamlSource.circuits = globalconfig.YamlSourceNone })

	in := make(chan util.Param, 2)
	in <- util.Param{Key: keys.Circuits, Val: map[string]int{"main": 1}}
	in <- util.Param{Key: keys.Circuits, Val: globalconfig.ConfigStatus{YamlSource: globalconfig.YamlSourceDb}}
	close(in)

	out := circuitsSource(in)

	// site data is wrapped with the source, an already wrapped value passes through
	assert.Equal(t, globalconfig.ConfigStatus{Config: map[string]int{"main": 1}, YamlSource: globalconfig.YamlSourceFile}, (<-out).Val)
	assert.Equal(t, globalconfig.ConfigStatus{YamlSource: globalconfig.YamlSourceDb}, (<-out).Val)
}

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
