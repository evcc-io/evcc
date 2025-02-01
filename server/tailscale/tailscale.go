///go:build gokrazy

package tailscale

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/evcc-io/evcc/util"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
)

func Run(host string, port int) error {
	log := util.NewLogger("tailscale")
	if host == "" {
		host = "evcc"
	}

	lc := new(tailscale.LocalClient)

	w, err := lc.WatchIPNBus(context.Background(), ipn.NotifyWatchEngineUpdates|ipn.NotifyInitialState)
	if err != nil {
		return err
	}

	h := &handler{
		log: log,
		lc:  lc,
	}

	started := make(chan struct{})
	go h.watchState(w, started)

	<-started
	go h.completeStartup(host)

	cfg, err := lc.GetServeConfig(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("cfg: %+v\n", cfg)

	if cfg.Web == nil {
		cfg.Web = make(map[ipn.HostPort]*ipn.WebServerConfig)
	}

	cfg.Web[ipn.HostPort(strconv.Itoa(port))] = &ipn.WebServerConfig{
		Handlers: map[string]*ipn.HTTPHandler{
			"foo": {
				Proxy: fmt.Sprintf("http://localhost:%d", port),
			},
		},
	}

	if err := lc.SetServeConfig(context.Background(), cfg); err != nil {
		return err
	}

	// ln, err := s.Listen("tcp", ":443")
	// if err != nil {
	// 	return  err
	// }

	// ln = tls.NewListener(ln, &tls.Config{
	// 	GetCertificate: lc.GetCertificate,
	// })

	// go handleListener(ln, strconv.Itoa(port))

	return nil
}

func handleListener(ln net.Listener, port string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go func(downstream net.Conn) {
			defer downstream.Close()

			upstream, err := net.Dial("tcp", ":"+port)
			if err != nil {
				return
			}
			defer upstream.Close()

			wg := new(sync.WaitGroup)
			wg.Add(2)

			go copy(wg, upstream, downstream)
			go copy(wg, downstream, upstream)

			wg.Wait()
		}(conn)
	}
}

func copy(wg *sync.WaitGroup, from io.Reader, to io.Writer) {
	_, _ = io.Copy(to, from)
	wg.Done()
}
