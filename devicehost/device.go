package devicehost

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/devicehost/proto/pb"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
)

type config struct {
	Host       string
	Device     string
	Properties map[string]string
}

// device is a handle to a device instantiated on a host
type device struct {
	host *Host
	id   string
	caps []string
}

// newDevice instantiates a device on its host
func newDevice(ctx context.Context, class templates.Class, other map[string]any) (*device, error) {
	var cc config
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	h, err := byName(cc.Host)
	if err != nil {
		return nil, err
	}

	res, err := h.client.New(ctx, &pb.NewRequest{
		DeviceClass: class.String(),
		Type:        cc.Device,
		Properties:  cc.Properties,
	})
	if err != nil {
		return nil, err
	}

	return &device{host: h, id: res.GetId(), caps: res.GetCapabilities()}, nil
}

// capability returns the wire name of an api interface, e.g. "api.Meter"
func capability[T any]() string {
	return reflect.TypeFor[T]().String()
}

// methodName resolves an api interface method expression like api.Meter.CurrentPower
// into its capability and method name
func methodName(expr any) (string, string, error) {
	typ := reflect.TypeOf(expr)
	if typ == nil || typ.Kind() != reflect.Func || typ.NumIn() == 0 {
		return "", "", fmt.Errorf("not a method expression: %T", expr)
	}

	recv := typ.In(0)
	if recv.Kind() != reflect.Interface {
		return "", "", fmt.Errorf("receiver is not an interface: %T", expr)
	}

	// interface method signatures omit the receiver
	in := make([]reflect.Type, 0, typ.NumIn()-1)
	for i := 1; i < typ.NumIn(); i++ {
		in = append(in, typ.In(i))
	}
	out := make([]reflect.Type, 0, typ.NumOut())
	for i := range typ.NumOut() {
		out = append(out, typ.Out(i))
	}
	sig := reflect.FuncOf(in, out, typ.IsVariadic())

	var method string
	for i := range recv.NumMethod() {
		m := recv.Method(i)
		if m.Type != sig {
			continue
		}
		if method != "" {
			return "", "", fmt.Errorf("%s: %s and %s share signature %s", recv, method, m.Name, sig)
		}
		method = m.Name
	}

	if method == "" {
		return "", "", fmt.Errorf("%s has no method with signature %s", recv, sig)
	}

	return recv.String(), method, nil
}

// call invokes an api interface method on the device, e.g. api.Meter.CurrentPower
func call(d *device, expr any, args []any, ret ...any) error {
	capability, method, err := methodName(expr)
	if err != nil {
		return err
	}

	return d.invoke(capability, method, args, ret...)
}

// invoke calls a capability method, decoding the reply into the ret pointers
func (d *device) invoke(capability, method string, args []any, ret ...any) error {
	encoded := make([][]byte, 0, len(args))
	for _, arg := range args {
		b, err := json.Marshal(arg)
		if err != nil {
			return err
		}
		encoded = append(encoded, b)
	}

	res, err := d.host.client.Call(context.TODO(), &pb.CallRequest{
		Id:         d.id,
		Capability: capability,
		Method:     method,
		Args:       encoded,
	})
	if err != nil {
		return err
	}

	if len(res.GetRet()) < len(ret) {
		return fmt.Errorf("%s.%s: expected %d values, got %d", capability, method, len(ret), len(res.GetRet()))
	}

	for i, v := range ret {
		if err := json.Unmarshal(res.GetRet()[i], v); err != nil {
			return fmt.Errorf("%s.%s: %w", capability, method, err)
		}
	}

	return nil
}

// decorate applies the capabilities the host reports
func (d *device) decorate(c implement.Caps) {
	for _, name := range d.caps {
		if f, ok := capTable[name]; ok {
			f(d, c)
		}
	}
}
