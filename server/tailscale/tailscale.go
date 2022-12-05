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

	var status *ipnstate.Status
	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

LOOP:
	for {
		status, err = lc.Status(ctx)
		if err != nil {
			return "", err
		}

		switch status.BackendState {
		case NeedsMachineAuth:
			logr.INFO.Printf("needs machine auth: %+v", status)
			break LOOP

		case NeedsLogin:
			if status.AuthURL == "" {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			logr.INFO.Printf("needs login: %s", status.AuthURL)
			break LOOP

		case Running:
			var net string
			if tn := status.CurrentTailnet; tn != nil {
				net = "." + tn.MagicDNSSuffix
			}

			logr.INFO.Printf("url: https://%s ip: %v", s.Hostname+net, status.TailscaleIPs)
			break LOOP

		default:
			logr.ERROR.Println("status:", status.BackendState, status.AuthURL)
			time.Sleep(10 * time.Millisecond)
		}
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
