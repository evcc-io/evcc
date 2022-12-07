package tailscale

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/evcc-io/evcc/util"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"tailscale.com/tsnet"
)

const (
	NoState          = "NoState"
	NeedsLogin       = "NeedsLogin"
	NeedsMachineAuth = "NeedsMachineAuth"
	Running          = "Running"
)

func Run(host, authKey string, downstreamPort int) (string, error) {
	logr := util.NewLogger("tailscale")
	if host == "" {
		host = "evcc"
	}

	s := &tsnet.Server{
		Hostname: host,
		AuthKey:  authKey,
		Logf:     logr.TRACE.Printf,
	}

	if err := s.Start(); err != nil {
		return "", err
	}

	lc, err := s.LocalClient()
	if err != nil {
		return "", err
	}

	w, err := lc.WatchIPNBus(context.Background(), ipn.NotifyWatchEngineUpdates|ipn.NotifyInitialState)
	if err != nil {
		return "", err
	}

	done := make(chan struct{})
	go watch(w, done)

	go func() {
		<-done

		status, err := lc.Status(context.Background())
		if err != nil {
			return
		}

		var net string
		if tn := status.CurrentTailnet; tn != nil {
			net = "." + tn.MagicDNSSuffix
		}

		logr.INFO.Printf("url: https://%s ip: %v", s.Hostname+net, status.TailscaleIPs)
	}()

	ln, err := s.Listen("tcp", ":443")
	if err != nil {
		return "", err
	}

	ln = tls.NewListener(ln, &tls.Config{
		GetCertificate: lc.GetCertificate,
	})

	go handle(ln, strconv.Itoa(downstreamPort))

	return "", nil
}

func watch(w *tailscale.IPNBusWatcher, done chan struct{}) {
	var needsLogin bool

	for {
		n, err := w.Next()
		if err != nil {
			return
		}

		if n.State != nil {
			fmt.Println(n.State.String())

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
				fmt.Println("LoginFinished")
				needsLogin = false

			case n.BrowseToURL != nil && *n.BrowseToURL != "":
				fmt.Println("BrowseToURL:", *n.BrowseToURL)
			}
		}
	}
}

func handle(ln net.Listener, port string) {
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
