// Package devicehost connects evcc to a remote host exposing device types.
// The host describes each type's properties, evcc registers them as templates.
package devicehost

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/devicehost/proto/pb"
	"github.com/evcc-io/evcc/util/templates"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Type is the registry type of devices provided by a device host
const Type = "devicehost"

var (
	mu    sync.RWMutex
	hosts = make(map[string]*Host)
)

// Host is a remote provider of device types and device instances
type Host struct {
	name       string
	conn       *grpc.ClientConn
	client     pb.DeviceHostClient
	registered map[templates.Class][]string
}

// New connects to the device host at uri and registers its device types
func New(ctx context.Context, name, uri string) (*Host, error) {
	conn, err := grpc.NewClient(uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	h := &Host{
		name:       name,
		conn:       conn,
		client:     pb.NewDeviceHostClient(conn),
		registered: make(map[templates.Class][]string),
	}

	if err := h.register(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	mu.Lock()
	defer mu.Unlock()

	if _, ok := hosts[name]; ok {
		h.unregister()
		conn.Close()
		return nil, fmt.Errorf("duplicate device host: %s", name)
	}
	hosts[name] = h

	return h, nil
}

// Close unregisters the host's device types and closes the connection
func (h *Host) Close() error {
	mu.Lock()
	defer mu.Unlock()

	h.unregister()
	delete(hosts, h.name)

	return h.conn.Close()
}

func byName(name string) (*Host, error) {
	mu.RLock()
	defer mu.RUnlock()

	h, ok := hosts[name]
	if !ok {
		return nil, fmt.Errorf("device host not found: %s", name)
	}

	return h, nil
}

// register fetches the host's device types and adds them as templates
func (h *Host) register(ctx context.Context) error {
	res, err := h.client.Types(ctx, &pb.TypesRequest{})
	if err != nil {
		return err
	}

	for _, typ := range res.GetTypes() {
		class, err := templates.ClassString(typ.GetDeviceClass())
		if err != nil {
			return fmt.Errorf("%s: %w", typ.GetType(), err)
		}

		tmpl := h.template(typ)
		if err := templates.Add(class, tmpl); err != nil {
			return err
		}

		h.registered[class] = append(h.registered[class], tmpl.Template)
	}

	return nil
}

func (h *Host) unregister() {
	for class, names := range h.registered {
		for _, name := range names {
			templates.Remove(class, name)
		}
	}
	clear(h.registered)
}

// templateName returns the evcc template name of a host device type
func templateName(host, typ string) string {
	return host + "-" + typ
}

// template converts a device type into an evcc template
func (h *Host) template(typ *pb.DeviceType) templates.Template {
	params := make([]templates.Param, 0, len(typ.GetProperties()))
	render := []string{
		"type: " + Type,
		"host: " + h.name,
		"device: " + typ.GetType(),
		"properties:",
	}

	for _, p := range typ.GetProperties() {
		params = append(params, param(p))
		// values are yaml-quoted by the renderer
		render = append(render, fmt.Sprintf("  %s: {{ .%s }}", p.GetName(), p.GetName()))
	}

	return templates.Template{
		Template: templateName(h.name, typ.GetType()),
		Params:   params,
		Render:   strings.Join(render, "\n") + "\n",
	}
}

var paramTypes = map[pb.PropertyType]templates.ParamType{
	pb.PropertyType_PROPERTY_TYPE_STRING:   templates.TypeString,
	pb.PropertyType_PROPERTY_TYPE_BOOL:     templates.TypeBool,
	pb.PropertyType_PROPERTY_TYPE_INT:      templates.TypeInt,
	pb.PropertyType_PROPERTY_TYPE_FLOAT:    templates.TypeFloat,
	pb.PropertyType_PROPERTY_TYPE_DURATION: templates.TypeDuration,
	pb.PropertyType_PROPERTY_TYPE_CHOICE:   templates.TypeChoice,
}

// param converts a host property into a template param
func param(p *pb.Property) templates.Param {
	return templates.Param{
		Name:        p.GetName(),
		Type:        paramTypes[p.GetType()],
		Description: templates.TextLanguage{Generic: p.GetTitle()},
		Help:        templates.TextLanguage{Generic: p.GetHelp()},
		Required:    p.GetRequired(),
		Mask:        p.GetMask(),
		Advanced:    p.GetAdvanced(),
		Default:     p.GetDefaultValue(),
		Example:     p.GetExample(),
		Unit:        p.GetUnit(),
		Choice:      p.GetChoice(),
	}
}
