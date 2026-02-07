package push

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	reg "github.com/evcc-io/evcc/util/registry"
)

// Messenger implements message sending
type Messenger interface {
	Send(title, msg string)
}

var registry = reg.New[Messenger]("messenger")

// NewFromConfig creates messenger from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (Messenger, error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, other)
	if err != nil {
		err = fmt.Errorf("cannot create messenger type '%s': %w", util.TypeWithTemplateName(typ, other), err)
	}

	return v, err
}
