///go:build gokrazy

package tailscale

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
)

type handler struct {
	log *util.Logger
	lc  *tailscale.LocalClient
}

func (h *handler) watchState(w *tailscale.IPNBusWatcher, done chan struct{}) {
	defer w.Close()

	var needsLogin bool
	for {
		n, err := w.Next()
		if err != nil {
			h.log.ERROR.Println(err)
			return
		}

		if n.State != nil {
			h.log.DEBUG.Println("state:", n.State.String())

			switch *n.State {
			case ipn.NeedsLogin:
				needsLogin = true

			case ipn.Running:
				close(done)
				return
			}
		}

		if needsLogin {
			switch {
			case n.LoginFinished != nil:
				h.log.DEBUG.Println("login finished")
				needsLogin = false

			case n.BrowseToURL != nil && *n.BrowseToURL != "":
				h.log.DEBUG.Println("login url:", *n.BrowseToURL)
			}
		}
	}
}

func (h *handler) completeStartup(hostname string) {
	status, err := h.lc.Status(context.Background())
	if err != nil {
		h.log.ERROR.Println(err)
		return
	}

	var net string
	if tn := status.CurrentTailnet; tn != nil {
		net = "." + tn.MagicDNSSuffix
	}

	h.log.INFO.Printf("url: https://%s ip: %v", hostname+net, status.TailscaleIPs)
}
