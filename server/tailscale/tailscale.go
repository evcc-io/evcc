///go:build gokrazy

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
)

func Run(port int) error {
	log := util.NewLogger("tailscale")
	lc := new(tailscale.LocalClient)

	w, err := lc.WatchIPNBus(context.Background(), ipn.NotifyWatchEngineUpdates|ipn.NotifyInitialState)
	if err != nil {
		return err
	}

	started := make(chan struct{})
	go func() {
		defer w.Close()

		var needsLogin bool
		for {
			n, err := w.Next()
			if err != nil {
				log.ERROR.Println(err)
				return
			}

			if n.State != nil {
				log.DEBUG.Println("state:", n.State.String())

				switch *n.State {
				case ipn.NeedsLogin:
					needsLogin = true

				case ipn.Running:
					close(started)
					return
				}
			}

			if needsLogin {
				switch {
				case n.LoginFinished != nil:
					log.DEBUG.Println("login finished")
					needsLogin = false

				case n.BrowseToURL != nil && *n.BrowseToURL != "":
					log.DEBUG.Println("login url:", *n.BrowseToURL)
				}
			}
		}
	}()

	<-started

	st, err := lc.StatusWithoutPeers(context.Background())
	if err != nil {
		return err
	}
	if st.Self == nil {
		return errors.New("no self node")
	}
	dnsName := strings.TrimSuffix(st.Self.DNSName, ".")

	sc, err := lc.GetServeConfig(context.Background())
	if err != nil {
		return err
	}

	sc.SetWebHandler(&ipn.HTTPHandler{
		Proxy: fmt.Sprintf("http://localhost:%d", port),
	}, dnsName, 443, "/", true)

	return lc.SetServeConfig(context.Background(), sc)
}
