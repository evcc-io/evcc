///go:build gokrazy

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
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

	sc, err := lc.GetServeConfig(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("host", host, "port", port)
	fmt.Printf("old sc: %+v\n", sc)

	// proxyPort := ":8443"

	// t, err := ipn.ExpandProxyTargetValue("localhost:80", []string{"http", "https", "https+insecure"}, "http")
	t, err := ipn.ExpandProxyTargetValue("http://localhost:80", []string{"http", "https", "https+insecure"}, "http")
	if err != nil {
		return err
	}

	st, err := lc.StatusWithoutPeers(context.Background())
	if err != nil {
		return err
	}
	if st.Self == nil {
		return errors.New("no self node")
	}
	dnsName := strings.TrimSuffix(st.Self.DNSName, ".")
	_ = dnsName
	_ = t

	sc.SetWebHandler(&ipn.HTTPHandler{
		Proxy: t,
	}, dnsName, 80, "", false)
	sc.SetWebHandler(&ipn.HTTPHandler{
		Proxy: t,
	}, dnsName, 443, "", true)

	fmt.Printf("new sc: %+v\n", sc)
	fmt.Println("dnsName", dnsName)

	if err := lc.SetServeConfig(context.Background(), sc); err != nil {
		return err
	}

	// ln, err := net.Listen("tcp", proxyPort)
	// if err != nil {
	// 	return err
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

		fmt.Println("!! CONNECTED")

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
