package tailscale

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"tailscale.com/tsnet"
)

const NoState = "NoState"

func Up(host, authKey string) error {
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
		return err
	}

	lc, err := s.LocalClient()
	if err != nil {
		return err
	}

	connect := time.Now()
	for {
		status, err := lc.Status(context.Background())
		if err != nil {
			return err
		}

		if status.BackendState == NoState {
			if time.Since(connect) > request.Timeout {
				return api.ErrTimeout
			}

			time.Sleep(10 * time.Millisecond)
			continue
		}

		var net string
		if tn := status.CurrentTailnet; tn != nil {
			net = "." + tn.MagicDNSSuffix
		}

		logr.INFO.Printf("url: http://%s ip: %v", s.Hostname+net, status.TailscaleIPs)
		break
	}

	ln, err := s.Listen("tcp", ":80")
	if err != nil {
		return err
	}

	// ln = tls.NewListener(ln, &tls.Config{
	// 	GetCertificate: lc.GetCertificate,
	// })

	go handle(ln)

	return nil
}

func handle(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			// logr.ERROR.Println(err)
			continue
		}

		go func(downstream net.Conn) {
			defer downstream.Close()

			upstream, err := net.Dial("tcp", ":7070")
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
