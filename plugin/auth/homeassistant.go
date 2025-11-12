package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/libp2p/zeroconf/v2"
	"golang.org/x/oauth2"
)

// https://developers.home-assistant.io/docs/auth_api

func init() {
	registry.AddCtx("homeassistant", NewHomeAssistantFromConfig)

	go scan(context.Background())
}

type HomeAssistantInstance struct {
	URI string
	oauth2.TokenSource
}

var (
	haMu        sync.Mutex
	haInstances = make(map[string]*HomeAssistantInstance)
)

func scan(ctx context.Context) {
	log := util.NewLogger("homeassistant")
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

				if err := authorize(se.Instance, uri); err != nil {
					log.ERROR.Println(err)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	if err := zeroconf.Browse(ctx, "_home-assistant._tcp.", "local.", entries); err != nil {
		log.ERROR.Println("zeroconf: failed to browse:", err.Error())
	}
}

func authorize(name, uri string) error {
	haMu.Lock()
	defer haMu.Unlock()

	if _, ok := haInstances[name]; ok {
		return nil
	}

	ts, err := NewHomeAssistant(context.Background(), name, uri)
	if err == nil {
		haInstances[name] = &HomeAssistantInstance{
			URI:         uri,
			TokenSource: ts,
		}
	}

	return err
}

func HomeAssistantInstanceNyName(name string) *HomeAssistantInstance {
	haMu.Lock()
	defer haMu.Unlock()
	return haInstances[name]
}

func NewHomeAssistantFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		URI  string
		Name string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeAssistant(ctx, cc.Name, cc.URI)
}

func NewHomeAssistant(ctx context.Context, name, uri string) (*OAuth, error) {
	localUri := "http://localhost:7070"
	redirectUri := localUri + "/providerauth/callback"

	log := util.NewLogger("homeassistant")
	ctx = context.WithValue(ctx, oauth2.HTTPClient, request.NewClient(log))

	oc := oauth2.Config{
		ClientID:    localUri,
		RedirectURL: redirectUri,
		Endpoint: oauth2.Endpoint{
			AuthURL:  uri + "/auth/authorize",
			TokenURL: uri + "/auth/token",
		},
	}

	return NewOauth(ctx, "HomeAssistant", name, &oc)
}
