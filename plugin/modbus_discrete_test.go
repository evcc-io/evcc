package plugin

import (
	"net"
	"testing"

	"github.com/andig/mbserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type discreteInputHandler struct {
	mbserver.DummyHandler
	inputs []bool
}

func (h *discreteInputHandler) HandleDiscreteInputs(req *mbserver.DiscreteInputsRequest) ([]bool, error) {
	if int(req.Addr)+int(req.Quantity) > len(h.inputs) {
		return nil, mbserver.ErrIllegalDataAddress
	}
	return h.inputs[req.Addr : req.Addr+req.Quantity], nil
}

func TestModbusDiscreteInput(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	srv, err := mbserver.New(&discreteInputHandler{inputs: []bool{false, true, false, true}})
	require.NoError(t, err)
	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	for addr, want := range []bool{false, true, false, true} {
		p, err := NewModbusFromConfig(t.Context(), map[string]any{
			"uri": l.Addr().String(),
			"id":  1,
			"register": map[string]any{
				"address":  addr,
				"type":     "discreteinput",
				"encoding": "bool8",
			},
		})
		require.NoError(t, err)

		g, err := p.(BoolGetter).BoolGetter()
		require.NoError(t, err)

		res, err := g()
		require.NoError(t, err)
		assert.Equal(t, want, res, addr)
	}
}
