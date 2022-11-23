package tailscale

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"tailscale.com/tsnet"
)

const NoState = "NoState"

func Up(host, authKey string) error {
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

		log.INFO.Printf("url: http://%s ip: %v", s.Hostname+net, status.TailscaleIPs)
		break
	}

	ln, err := s.Listen("tcp", ":80")
	if err != nil {
		return err
	}

	// if *addr == ":443" {
	// 	ln = tls.NewListener(ln, &tls.Config{
	// 		GetCertificate: tailscale.GetCertificate,
	// 	})
	// }

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.ERROR.Println(err)
				continue
			}

			fmt.Println("Accept")

			go func(downstream net.Conn) {
				defer downstream.Close()

				upstream, err := net.Dial("tcp", ":7070")
				if err != nil {
					return
				}
				defer upstream.Close()

				fmt.Println("Upstream")

				_, _ = io.Copy(upstream, downstream)
				_, _ = io.Copy(downstream, upstream)
			}(conn)
		}
	}()

	// go func() {
	// 	log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 		who, err := lc.WhoIs(r.Context(), r.RemoteAddr)
	// 		if err != nil {
	// 			http.Error(w, err.Error(), 500)
	// 			return
	// 		}

	// 		fmt.Fprintf(w, "<html><body><h1>Hello, world!</h1>\n")
	// 		fmt.Fprintf(w, "<p>You are <b>%s</b> from <b>%s</b> (%s)</p>",
	// 			html.EscapeString(who.UserProfile.LoginName),
	// 			html.EscapeString(firstLabel(who.Node.ComputedName)),
	// 			r.RemoteAddr)
	// 	})))
	// }()

	return nil
}

func firstLabel(s string) string {
	s, _, _ = strings.Cut(s, ".")
	return s
}
