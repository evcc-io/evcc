package tariff

import (
	"crypto/sha256"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
)

type cached struct {
	Type  api.TariffType `json:"type"`
	Rates api.Rates      `json:"rates"`
}

func cacheKey(typ string, other map[string]any) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s-%v", typ, other))))
}

func cachePut(key string, typ api.TariffType, rates api.Rates) error {
	return settings.SetJson(key, &cached{
		Type:  api.TariffType(typ),
		Rates: rates,
	})
}

func cacheGet(key string) (*cached, error) {
	var res cached
	return &res, settings.Json(key, &res)
}
