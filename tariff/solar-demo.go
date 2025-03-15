package tariff

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

//go:embed solar-demo.json
var demoJson string

func init() {
	registry.AddCtx("solar-demo", NewDemoFromConfig)
}

func NewDemoFromConfig(ctx context.Context, other map[string]interface{}) (api.Tariff, error) {
	t := &Tariff{
		log:   util.NewLogger("demo"),
		embed: new(embed),
		typ:   api.TariffTypeSolar,
		data:  util.NewMonitor[api.Rates](2 * time.Hour),
	}

	var err error
	done := make(chan error)
	go t.run(func() (string, error) {
		s := demoJson
		now := time.Now()

		for i := 2; i >= 0; i-- {
			date := time.Date(now.Year(), now.Month(), now.Day()+i, 0, 0, 0, 0, time.Local)
			s = strings.ReplaceAll(s, fmt.Sprintf(`2025-03-%02d`, 8+i), date.Format("2006-01-02"))
		}

		return s, nil
	}, done, time.Hour)
	err = <-done

	return t, err
}
