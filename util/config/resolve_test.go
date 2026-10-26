package config

import (
	"context"
	"errors"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefDecode(t *testing.T) {
	var cc struct {
		Name   Ref
		Inline Ref
		Opt    *Ref
	}
	require.NoError(t, util.DecodeOther(map[string]any{
		"name":   "db:1",
		"inline": map[string]any{"type": "const", "value": 2},
		"opt":    "db:2",
	}, &cc))
	assert.Equal(t, Ref{Name: "db:1"}, cc.Name)
	assert.Equal(t, Ref{Type: "const", Other: map[string]any{"value": 2}}, cc.Inline)
	assert.Equal(t, &Ref{Name: "db:2"}, cc.Opt)
}

func TestResolve(t *testing.T) {
	// device instances are interfaces, a disabled device holds a nil interface
	h := &handler[any]{topic: "test"}
	one := 1
	require.NoError(t, h.Add(NewStaticDevice[any](Named{Name: "one"}, &one)))
	require.NoError(t, h.Add(NewStaticDevice[any](Named{Name: "disabled"}, nil)))

	newFromConf := func(_ context.Context, typ string, other map[string]any) (any, error) {
		if typ != "const" {
			return nil, errors.New("unexpected type")
		}
		v := other["value"].(int)
		return &v, nil
	}

	ctx := context.Background()

	res, err := Resolve(ctx, h, Ref{Name: "one"}, newFromConf)
	require.NoError(t, err)
	assert.Same(t, &one, res, "reference returns the configured instance")

	_, err = Resolve(ctx, h, Ref{Name: "disabled"}, newFromConf)
	require.Error(t, err, "disabled device has no instance")

	_, err = Resolve(ctx, h, Ref{Name: "missing"}, newFromConf)
	require.ErrorIs(t, err, ErrNotFound)

	res, err = Resolve(ctx, h, Ref{Type: "const", Other: map[string]any{"value": 2}}, newFromConf)
	require.NoError(t, err)
	assert.Equal(t, 2, *res.(*int), "inline config is instantiated")

	_, err = Resolve(ctx, h, Ref{}, newFromConf)
	require.Error(t, err, "neither name nor config")
}
