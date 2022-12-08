package tailscale

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/evcc-io/evcc/util"
	"tailscale.com/ipn"
	"tailscale.com/tsnet"
)

func Run(host, authKey string, port int) (string, error) {
	log := util.NewLogger("tailscale")
	if host == "" {
		host = "evcc"
	}

	s := &tsnet.Server{
		Hostname: host,
		AuthKey:  authKey,
		Logf:     log.TRACE.Printf,
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

	h := &handler{
		log: log,
		lc:  lc,
	}

	started := make(chan struct{})
	go h.watchState(w, started)

	<-started
	go h.completeStartup(host)

	ln, err := s.Listen("tcp", ":443")
	if err != nil {
		return "", err
	}

	ln = tls.NewListener(ln, &tls.Config{
		GetCertificate: lc.GetCertificate,
	})

	go handleListener(ln, strconv.Itoa(port))

	return "", nil
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
