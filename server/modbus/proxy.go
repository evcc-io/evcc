package modbus

import (
	"fmt"
	"net"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

func StartProxy(port int, config modbus.Settings, readOnly ReadOnlyMode) error {
	conn, err := modbus.NewConnection(config.URI, config.Device, config.Comset, config.Baudrate, modbus.ProtocolFromRTU(config.RTU), config.ID)
	if err != nil {
		return err
	}

	if !sponsor.IsAuthorized() {
		return api.ErrSponsorRequired
	}

	h := &handler{
		log:      util.NewLogger(fmt.Sprintf("proxy-%d", port)),
		readOnly: readOnly,
		conn:     conn,
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	h.log.DEBUG.Printf("modbus proxy for %s listening at :%d", config.String(), port)

	srv, err := mbserver.New(h, mbserver.Logger(&logger{log: h.log}))

	if err == nil {
		err = srv.Start(l)
	}

	return err
}
