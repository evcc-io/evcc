package homeassistant

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/libp2p/zeroconf/v2"
)

var (
	mu        sync.Mutex
	instances = make(map[string]string)
)

func init() {
	go scan()
}

func instanceUriByName(name string) string {
	mu.Lock()
	defer mu.Unlock()
	return instances[name]
}

func instanceNameByUri(uri string) string {
	mu.Lock()
	defer mu.Unlock()

	for n, u := range instances {
		if uri == u {
			return n
		}
	}

	return ""
}

func addInstance(name, uri string) {
	mu.Lock()
	defer mu.Unlock()
	instances[name] = strings.TrimRight(uri, "/")
}

func scan() {
	ctx := context.Background()

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

				addInstance(se.Instance, uri)

			case <-ctx.Done():
				return
			}
		}
	}()

	if err := zeroconf.Browse(ctx, "_home-assistant._tcp.", "local.", entries); err != nil {
		log := util.NewLogger("homeassistant")
		log.ERROR.Println("zeroconf: failed to browse:", err.Error())
	}
}
