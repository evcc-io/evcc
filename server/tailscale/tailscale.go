package tailscale

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"tailscale.com/tsnet"
)

 func defStr(s string, dflt string) string {
				if s == "" {
					return dflt
				}
				return s
			}

func Up(host, authKey string) error {
	s := &tsnet.Server{
		Hostname: defStr(host,"evcc"),
		AuthKey:  authKey,
		Logf:     util.NewLogger("tailscale").ERROR.Printf,
	}
	// defer s.Close()

	ln, err := s.Listen("tcp", ":80")
	if err != nil {
		return err
	}
	defer ln.Close()

	lc, err := s.LocalClient()
	if err != nil {
		return err
	}

	// if *addr == ":443" {
	// 	ln = tls.NewListener(ln, &tls.Config{
	// 		GetCertificate: tailscale.GetCertificate,
	// 	})
	// }

	go func() {
		log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			who, err := lc.WhoIs(r.Context(), r.RemoteAddr)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			fmt.Fprintf(w, "<html><body><h1>Hello, world!</h1>\n")
			fmt.Fprintf(w, "<p>You are <b>%s</b> from <b>%s</b> (%s)</p>",
				html.EscapeString(who.UserProfile.LoginName),
				html.EscapeString(firstLabel(who.Node.ComputedName)),
				r.RemoteAddr)
		})))
	}()

	return nil
}

func firstLabel(s string) string {
	s, _, _ = strings.Cut(s, ".")
	return s
}
