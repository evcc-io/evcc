package ecoflow

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// portalManifest lists the bundles of the EcoFlow web portal module that
// carries the (appKey, secret) table.
const portalManifest = "https://cdn-fe.ecoflow.com/ef-device-user-jt/asset-manifest.json"

var (
	appKeyTable = regexp.MustCompile(`\[(\["[0-9a-f]{32}","[0-9a-f]{32}"\],?){8,}\]`)
	appKeyPair  = regexp.MustCompile(`\["([0-9a-f]{32})","([0-9a-f]{32})"\]`)
)

// AppKey is one (appKey, secret) pair of the EcoFlow web app. The broker only
// accepts client IDs signed with such a pair.
type AppKey struct {
	Key, Secret string
}

// AppKeys reads the (appKey, secret) pairs from the public web portal bundle,
// so that a rotation by EcoFlow needs no evcc release.
func AppKeys() ([]AppKey, error) {
	// request.Helper would trace the 2 MB bundle
	client := &request.Helper{Client: &http.Client{Timeout: request.Timeout, Transport: transport.Default()}}

	var manifest struct {
		Files map[string]string `json:"files"`
	}
	if err := client.GetJSON(portalManifest, &manifest); err != nil {
		return nil, fmt.Errorf("app keys: %w", err)
	}

	bundle, err := client.GetBody(manifest.Files["main.js"])
	if err != nil {
		return nil, fmt.Errorf("app keys: %w", err)
	}

	return parseAppKeys(bundle)
}

func parseAppKeys(bundle []byte) ([]AppKey, error) {
	var res []AppKey
	for _, m := range appKeyPair.FindAllSubmatch(appKeyTable.Find(bundle), -1) {
		res = append(res, AppKey{Key: string(m[1]), Secret: string(m[2])})
	}

	if len(res) == 0 {
		return nil, errors.New("app keys: no key table in web portal bundle")
	}

	return res, nil
}
