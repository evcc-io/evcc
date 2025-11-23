package homeassistant

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	"github.com/libp2p/zeroconf/v2"
)

func init() {
	network.Register(scan)
}

func scan() {
	ctx := context.Background()

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
	mu.Lock()
	defer mu.Unlock()

	if _, ok := instances[name]; ok {
		return nil
	}

	ts, err := NewHomeAssistant(name, uri)
	if err != nil {
		return err
	}

	instances[name] = &instance{
		URI:         uri,
		TokenSource: ts,
	}

	return nil
}
