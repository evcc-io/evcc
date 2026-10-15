package discovery

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
	"strings"
	"sync"
)

// update with `make oui-update`
//
//go:embed oui.txt.gz
var ouiData []byte

var vendors = sync.OnceValue(func() map[string]string {
	res := make(map[string]string)

	r, err := gzip.NewReader(bytes.NewReader(ouiData))
	if err != nil {
		return res
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return res
	}

	for line := range strings.Lines(string(b)) {
		if prefix, name, ok := strings.Cut(strings.TrimSpace(line), "\t"); ok {
			res[prefix] = name
		}
	}

	return res
})

// Registration returns the registered prefix and organization of the hardware address
func Registration(mac string) (string, string) {
	hex := strings.ToUpper(strings.NewReplacer(":", "", "-", "").Replace(mac))

	// longest assignment wins: MA-S, MA-M, MA-L
	for _, l := range []int{9, 7, 6} {
		if len(hex) < l {
			continue
		}
		if name, ok := vendors()[hex[:l]]; ok {
			return hex[:l], name
		}
	}

	return "", ""
}

// Vendor returns the registered organization of the hardware address
func Vendor(mac string) string {
	_, name := Registration(mac)
	return name
}
