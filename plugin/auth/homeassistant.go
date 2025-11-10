package auth

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/libp2p/zeroconf/v2"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("homeassistant", NewHomeAssistantFromConfig)

	go scan(context.Background())
}

func scan(ctx context.Context) {
	entries := make(chan *zeroconf.ServiceEntry, 1)

	go func() {
		for {
			select {
			case se := <-entries:
				uri := fmt.Sprintf("http://%s:%d", se.HostName, se.Port)

			OUTER:
				for _, text := range se.Text {
					for _, prefix := range []string{"external_url", "base_url", "internal_url"} {
						if u, ok := strings.CutPrefix(text, prefix+"="); ok && u != "" {
							uri = u
							break OUTER
						}
					}
				}

				go authorize(se.Instance, uri)

			case <-ctx.Done():
				return
			}
		}
	}()

	if err := zeroconf.Browse(ctx, "_home-assistant._tcp.", "local.", entries); err != nil {
		fmt.Println("zeroconf: failed to browse:", err.Error())
	}
}

func authorize(name, uri string) {
	if _, err := NewHomeAssistantFromConfig(context.Background(), map[string]any{
		"uri":  uri,
		"name": name,
	}); err != nil {
		log.Println(err)
	}
}

func NewHomeAssistantFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		URI  string
		Name string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	uri := "http://localhost:7070"
	redirectUri := uri + "/providerauth/callback"

	log := util.NewLogger("homeassistant")
	ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	oc := oauth2.Config{
		ClientID:    uri,
		RedirectURL: redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cc.URI + "/auth/authorize",
			TokenURL: cc.URI + "/auth/token",
		},
	}

	return NewOauth(ctx, "HomeAssistant", cc.Name, &oc)
}
