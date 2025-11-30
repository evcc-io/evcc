package tariff

import (
	"crypto/sha256"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/cache"
)

type cached struct {
	Type  api.TariffType `json:"type"`
	Rates api.Rates      `json:"rates"`
}

func cacheKey(typ string, other map[string]any) string {
	return fmt.Sprintf("%x", sha256.Sum256(fmt.Appendf(nil, "%s-%v", typ, other)))
}

func cachePut(key string, typ api.TariffType, rates api.Rates) error {
	return cache.Put(key, &cached{
		Type:  typ,
		Rates: rates,
	})
}

func cacheGet(key string) (*cached, error) {
	var res cached
	return &res, cache.Get(key, &res)
}
