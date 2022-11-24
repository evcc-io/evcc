package tailscale

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

const NoState = "NoState"

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

	var status *ipnstate.Status
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	for {
		status, err = lc.Status(ctx)
		if err != nil {
			return "", err
		}

		if status.BackendState == NoState {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		var net string
		if tn := status.CurrentTailnet; tn != nil {
			net = "." + tn.MagicDNSSuffix
		}

		logr.INFO.Printf("url: https://%s ip: %v", s.Hostname+net, status.TailscaleIPs)
		break
	}

	ln, err := s.Listen("tcp", ":443")
	if err != nil {
		return "", err
	}

	ln = tls.NewListener(ln, &tls.Config{
		GetCertificate: lc.GetCertificate,
	})

	go handle(ln, strconv.Itoa(downstreamPort))

	return status.AuthURL, nil
}

func handle(ln net.Listener, port string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			// logr.ERROR.Println(err)
			continue
		}

		go func(downstream net.Conn) {
			defer downstream.Close()

			upstream, err := net.Dial("tcp", ":"+port)
			if err != nil {
				return
			}
			defer upstream.Close()

			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				_, _ = io.Copy(upstream, downstream)
				wg.Done()
			}()
			go func() {
				_, _ = io.Copy(downstream, upstream)
				wg.Done()
			}()

			wg.Wait()
		}(conn)
	}
}
