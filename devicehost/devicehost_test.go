package devicehost_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/devicehost"
	"github.com/evcc-io/evcc/devicehost/proto/pb"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// exampleHost is a device host exposing a pv meter and a wallbox type
type exampleHost struct {
	pb.UnimplementedDeviceHostServer
	devices map[string]*exampleDevice
}

type exampleDevice struct {
	properties map[string]string
	enabled    bool
	current    int64
	phases     int
}

// capabilities are the optional api interfaces per device type
var capabilities = map[string][]string{
	"pv":      {"api.Meter", "api.MeterEnergy"},
	"wallbox": {"api.PhaseSwitcher"},
}

func uriProperty() *pb.Property {
	return &pb.Property{
		Name:     "uri",
		Title:    "URI",
		Type:     pb.PropertyType_PROPERTY_TYPE_STRING,
		Required: true,
	}
}

// Types describes the device types and their configuration properties
func (h *exampleHost) Types(context.Context, *pb.TypesRequest) (*pb.TypesReply, error) {
	return &pb.TypesReply{Types: []*pb.DeviceType{
		{
			DeviceClass: "meter",
			Type:        "pv",
			Title:       "Example PV Meter",
			Properties: []*pb.Property{
				uriProperty(),
				{
					Name:         "scale",
					Title:        "Scale",
					Type:         pb.PropertyType_PROPERTY_TYPE_FLOAT,
					DefaultValue: "1",
					Advanced:     true,
				},
			},
		},
		{
			DeviceClass: "charger",
			Type:        "wallbox",
			Title:       "Example Wallbox",
			Properties:  []*pb.Property{uriProperty()},
		},
	}}, nil
}

// New instantiates a device from the configured properties
func (h *exampleHost) New(_ context.Context, req *pb.NewRequest) (*pb.NewReply, error) {
	caps, ok := capabilities[req.GetType()]
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", req.GetType())
	}

	if req.GetProperties()["uri"] == "" {
		return nil, fmt.Errorf("%s: missing uri", req.GetType())
	}

	id := fmt.Sprintf("%s-%d", req.GetType(), len(h.devices))
	h.devices[id] = &exampleDevice{properties: req.GetProperties(), phases: 3}

	return &pb.NewReply{Id: id, Capabilities: caps}, nil
}

// reply json encodes the return values
func reply(values ...any) (*pb.CallReply, error) {
	res := new(pb.CallReply)

	for _, v := range values {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		res.Ret = append(res.Ret, b)
	}

	return res, nil
}

// Call invokes a capability method on a device instance
func (h *exampleHost) Call(_ context.Context, req *pb.CallRequest) (*pb.CallReply, error) {
	dev, ok := h.devices[req.GetId()]
	if !ok {
		return nil, fmt.Errorf("unknown device: %s", req.GetId())
	}

	arg := func(i int, v any) error {
		return json.Unmarshal(req.GetArgs()[i], v)
	}

	switch req.GetCapability() + "." + req.GetMethod() {
	case "api.Meter.CurrentPower":
		scale, err := strconv.ParseFloat(dev.properties["scale"], 64)
		if err != nil {
			return nil, err
		}
		return reply(-3000 * scale)

	case "api.MeterEnergy.TotalEnergy":
		return reply(1234.5)

	case "api.ChargeState.Status":
		if dev.enabled {
			return reply(api.StatusC)
		}
		return reply(api.StatusA)

	case "api.Charger.Enabled":
		return reply(dev.enabled)

	case "api.Charger.Enable":
		return new(pb.CallReply), arg(0, &dev.enabled)

	case "api.CurrentController.MaxCurrent":
		return new(pb.CallReply), arg(0, &dev.current)

	case "api.PhaseSwitcher.Phases1p3p":
		return new(pb.CallReply), arg(0, &dev.phases)

	default:
		return nil, fmt.Errorf("unknown method: %s.%s", req.GetCapability(), req.GetMethod())
	}
}

// serve starts the example device host on an ephemeral port
func serve(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	pb.RegisterDeviceHostServer(srv, &exampleHost{devices: make(map[string]*exampleDevice)})

	go func() { _ = srv.Serve(l) }()
	t.Cleanup(srv.Stop)

	return l.Addr().String()
}

func TestMeter(t *testing.T) {
	ctx := context.Background()

	host, err := devicehost.New(ctx, "demo", serve(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = host.Close() })

	// the host's device type is registered as a template
	tmpl, err := templates.ByName(templates.Meter, "demo-pv")
	require.NoError(t, err)

	i, uri := tmpl.ParamByName("uri")
	require.NotEqual(t, -1, i)
	assert.True(t, uri.Required)
	assert.Equal(t, templates.TypeString, uri.Type)

	i, scale := tmpl.ParamByName("scale")
	require.NotEqual(t, -1, i)
	assert.Equal(t, templates.TypeFloat, scale.Type)
	assert.Equal(t, "1", scale.Default)

	// instantiate through the regular configuration path
	m, err := meter.NewFromConfig(ctx, "template", map[string]any{
		"template": "demo-pv",
		"uri":      "http://demo/pv",
		"scale":    2,
	})
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, -6000.0, power)

	// optional capability reported by the host
	me, ok := api.Cap[api.MeterEnergy](m)
	require.True(t, ok)

	energy, err := me.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 1234.5, energy)

	// capability not reported by the host
	assert.False(t, api.HasCap[api.Battery](m))

	// closing the host removes its device types again
	require.NoError(t, host.Close())

	_, err = templates.ByName(templates.Meter, "demo-pv")
	assert.Error(t, err)
}

func TestCharger(t *testing.T) {
	ctx := context.Background()

	host, err := devicehost.New(ctx, "demo3", serve(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = host.Close() })

	c, err := charger.NewFromConfig(ctx, "template", map[string]any{
		"template": "demo3-wallbox",
		"uri":      "http://demo/wb",
	})
	require.NoError(t, err)

	status, err := c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusA, status)

	require.NoError(t, c.Enable(true))
	require.NoError(t, c.MaxCurrent(16))

	enabled, err := c.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	status, err = c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	ps, ok := api.Cap[api.PhaseSwitcher](c)
	require.True(t, ok)
	require.NoError(t, ps.Phases1p3p(1))
}

func TestMissingRequiredProperty(t *testing.T) {
	ctx := context.Background()

	host, err := devicehost.New(ctx, "demo2", serve(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = host.Close() })

	_, err = meter.NewFromConfig(ctx, "template", map[string]any{
		"template": "demo2-pv",
	})
	assert.Error(t, err)
}
